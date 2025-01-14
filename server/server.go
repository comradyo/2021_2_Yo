package server

import (
	log "backend/pkg/logger"
	"backend/pkg/register"
	"backend/pkg/utils"
	authDelivery "backend/service/auth/delivery/http"
	"errors"

	userRepository "backend/microservice/user/proto"
	userDelivery "backend/service/user/delivery/http"
	userUseCase "backend/service/user/usecase"

	eventRepository "backend/microservice/event/proto"
	eventDelivery "backend/service/event/delivery/http"
	eventUseCase "backend/service/event/usecase"

	"backend/middleware"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	sql "github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	protoAuth "backend/microservice/auth/proto"
	authUseCase "backend/service/auth/usecase"

	"backend/prometheus"
)

const logMessage = "server:"

type Options struct {
	LogLevel logrus.Level
	Testing  bool
}

type App struct {
	Options      *Options
	AuthManager  *authDelivery.Delivery
	UserManager  *userDelivery.Delivery
	EventManager *eventDelivery.Delivery
	db           *sql.DB
}

func NewApp(opts *Options) (*App, error) {
	if opts == nil {
		return nil, errors.New("Unexpected NewApp error")
	}
	message := logMessage + "NewApp:"
	log.Init(opts.LogLevel)
	log.Info(fmt.Sprintf(message+"started, log level = %s", opts.LogLevel))

	db, err := utils.InitPostgresDB()
	if err != nil {
		log.Error(message+"err = ", err)
		if !opts.Testing {
			return nil, err
		}
	}

	authPort := viper.GetString("auth_port")
	authHost := viper.GetString("auth_host")
	AuthAddr := authHost + ":" + authPort

	grpcConnAuth, err := grpc.Dial(
		AuthAddr,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Error(message+"err = ", err)
		if !opts.Testing {
			return nil, err
		}
	}

	authClient := protoAuth.NewAuthClient(grpcConnAuth)
	authService := authUseCase.NewUseCase(authClient)
	authD := authDelivery.NewDelivery(authService)

	userPort := viper.GetString("user_port")
	userHost := viper.GetString("user_host")
	userMicroserviceAddr := userHost + ":" + userPort

	userGrpcConn, err := grpc.Dial(userMicroserviceAddr, grpc.WithInsecure())
	if err != nil {
		log.Error(message+"err = ", err)
		if !opts.Testing {
			return nil, err
		}
	}

	userR := userRepository.NewRepositoryClient(userGrpcConn)
	userUC := userUseCase.NewUseCase(userR)
	userD := userDelivery.NewDelivery(userUC)

	eventPort := viper.GetString("event_port")
	eventHost := viper.GetString("event_host")
	eventMicroserviceAddr := eventHost + ":" + eventPort

	eventGrpcConn, err := grpc.Dial(eventMicroserviceAddr, grpc.WithInsecure())
	if err != nil {
		log.Error(message+"err = ", err)
		if !opts.Testing {
			return nil, err
		}
	}

	eventR := eventRepository.NewRepositoryClient(eventGrpcConn)
	eventUC := eventUseCase.NewUseCase(eventR)
	eventD := eventDelivery.NewDelivery(eventUC)

	return &App{
		Options:      opts,
		AuthManager:  authD,
		UserManager:  userD,
		EventManager: eventD,
		db:           db,
	}, nil
}

func options(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	log.Debug("options: ", cookie)
}

func newRouterWithEndpoints(app *App) *mux.Router {
	mw := middleware.NewMiddlewares(app.AuthManager.UseCase)
	mm := prometheus.NewMetricsMiddleware()

	r := mux.NewRouter()
	r.Use(mw.GetVars)
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Recovery)
	r.Use(mm.Metrics)
	r.Methods("OPTIONS").HandlerFunc(options)

	//TODO: Потом раскоментить и убрать то, что снизу
	//authRouter := r.PathPrefix("/auth").Subrouter()
	//register.AuthHTTPEndpoints(authRouter, app.AuthManager, mw)

	r.HandleFunc("/auth/signup", app.AuthManager.SignUp).Methods("POST")
	r.HandleFunc("/auth/login", app.AuthManager.SignIn).Methods("POST")
	logoutHandlerFunc := http.HandlerFunc(app.AuthManager.Logout)
	r.Handle("/auth/logout", mw.Auth(logoutHandlerFunc))

	eventRouter := r.PathPrefix("/events").Subrouter()
	eventRouter.Methods("POST").Subrouter().Use(mw.CSRF)

	register.EventHTTPEndpoints(eventRouter, app.EventManager, mw)

	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Methods("POST").Subrouter().Use(mw.CSRF)
	register.UserHTTPEndpoints(userRouter, app.UserManager, app.EventManager, mw)

	r.Handle("/metrics", promhttp.Handler())

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
		port = viper.GetString("bmstusa_port")
	}
	log.Info(message+"port =", port)
	if app.Options.Testing {
		port = "wrong port"
	}
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Error(message+"err = ", err)
		return err
	}
	return nil
}
