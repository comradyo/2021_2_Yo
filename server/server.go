package server

import (
	authDelivery "backend/service/auth/delivery/http"
	authRepository "backend/service/auth/repository/postgres"
	authUseCase "backend/service/auth/usecase"

	userDelivery "backend/service/user/delivery/http"
	userRepository "backend/service/user/repository/postgres"
	userUseCase "backend/service/user/usecase"

	eventDelivery "backend/service/event/delivery/http"
	eventRepository "backend/service/event/repository/postgres"
	eventUseCase "backend/service/event/usecase"

	session "backend/service/session/manager"
	sessionRepository "backend/service/session/repository"

	"backend/register"

	log "backend/logger"
	"backend/middleware"
	"backend/utils"
	"fmt"
	"net/http"
	"os"

	"backend/service/csrf/manager"
	csrfMiddleware "backend/service/csrf/middleware"
	csrfRepository "backend/service/csrf/repository"

	"github.com/gorilla/mux"
	sql "github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

const logMessage = "server:"

type App struct {
	AuthManager    *authDelivery.Delivery
	UserManager    *userDelivery.Delivery
	EventManager   *eventDelivery.Delivery
	SessionManager *session.Manager
	csrfManager    *csrf.Manager
	db             *sql.DB
}

func NewApp(logLevel logrus.Level) (*App, error) {
	message := logMessage + "NewApp:"
	log.Init(logLevel)
	log.Info(fmt.Sprintf(message+"started, log level = %s", logLevel))

	secret, err := utils.GetSecret()
	if err != nil {
		log.Error(message+"err =", err)
		return nil, err
	}
	db, err := utils.InitPostgresDB()
	if err != nil {
		log.Error(message+"err =", err)
		return nil, err
	}
	redisDB, err := utils.InitRedisDB("redis_db_session")
	if err != nil {
		log.Error(message+"err =", err)
		return nil, err
	}

	redisConnCSRFTokens, err := utils.InitRedisDB("redis_db_csrf")
	if err != nil {
		log.Error(message+"err =", err)
		return nil, err
	}

	sessionR := sessionRepository.NewRepository(redisDB)
	sessionM := session.NewManager(*sessionR)

	csrfR := csrfRepository.NewRepository(redisConnCSRFTokens)
	csrfM := csrf.NewManager(*csrfR)

	authR := authRepository.NewRepository(db)
	authUC := authUseCase.NewUseCase(authR, []byte(secret))
	authD := authDelivery.NewDelivery(authUC, sessionM, csrfM)

	userR := userRepository.NewRepository(db)
	userUC := userUseCase.NewUseCase(userR)
	userD := userDelivery.NewDelivery(userUC,csrfM)

	eventR := eventRepository.NewRepository(db)
	eventUC := eventUseCase.NewUseCase(eventR)
	eventD := eventDelivery.NewDelivery(eventUC)

	return &App{
		AuthManager:    authD,
		UserManager:    userD,
		EventManager:   eventD,
		SessionManager: sessionM,
		db:             db,
	}, nil
}

func options(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	log.Debug("options: ", cookie)
}

func newRouterWithEndpoints(app *App) *mux.Router {
	mw := middleware.NewMiddlewares(app.SessionManager)
	csrfMW := csrfMiddleware.NewMiddleware(app.csrfManager)

	r := mux.NewRouter()
	r.Use(mw.GetVars)
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Recovery)
	r.Methods("OPTIONS").HandlerFunc(options)

	//TODO: Потом раскоментить и убрать то, что снизу
	//authRouter := r.PathPrefix("/auth").Subrouter()
	//register.AuthHTTPEndpoints(authRouter, app.AuthManager, mw)

	r.HandleFunc("/auth/signup", app.AuthManager.SignUp).Methods("POST")
	r.HandleFunc("/auth/login", app.AuthManager.SignIn).Methods("POST")
	logoutHandlerFunc := http.HandlerFunc(app.AuthManager.Logout)
	r.Handle("/auth/logout", mw.Auth(logoutHandlerFunc))

	eventRouter := r.PathPrefix("/events").Subrouter()
	eventRouter.Methods("POST").Subrouter().Use(csrfMW.CSRF)
	
	register.EventHTTPEndpoints(eventRouter, app.EventManager, mw)

	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Methods("POST").Subrouter().Use(csrfMW.CSRF)
	//getUserHandlerFunc := mw.Auth(http.HandlerFunc(app.UserManager.GetUser))
	//r.Handle("/user", getUserHandlerFunc).Methods("GET")
	register.UserHTTPEndpoints(userRouter, app.UserManager, mw)


	return r
}

func (app *App) Run() error {
	if app.db != nil {
		defer app.db.Close()
	}
	message := logMessage + "Run:"
	log.Info(message + "start")
	r := newRouterWithEndpoints(app)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Info(message+"port =", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Error(message+"err =", err)
		return err
	}
	return nil
}
