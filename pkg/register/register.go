package register

import (
	"backend/middleware"
	log "backend/pkg/logger"
	authHttp "backend/service/auth/delivery/http"
	eventHttp "backend/service/event/delivery/http"
	userHttp "backend/service/user/delivery/http"
	"github.com/gorilla/mux"
	"net/http"
)

func useMiddlewares(r *mux.Router, path string, handlerFunc http.HandlerFunc, middlewares ...mux.MiddlewareFunc) *mux.Router {
	result := r.NewRoute().Subrouter()
	result.HandleFunc(path, handlerFunc)
	for _, mw := range middlewares {
		result.Use(mw)
	}
	log.Debug("useMiddlewares ended")
	return result
}

func AuthHTTPEndpoints(r *mux.Router, delivery *authHttp.Delivery, middlewares *middleware.Middlewares) {
	r.HandleFunc("/signup", delivery.SignUp)
	r.HandleFunc("/login", delivery.SignIn)
	logoutHandlerFunc := http.HandlerFunc(delivery.Logout)
	r.Handle("/logout", middlewares.Auth(logoutHandlerFunc))
}

func UserHTTPEndpoints(r *mux.Router, uDelivery *userHttp.Delivery, eDelivery *eventHttp.Delivery, mws *middleware.Middlewares) {
	r.HandleFunc("/{id:[0-9]+}", uDelivery.GetUserById).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/events/favourite", eDelivery.GetVisitedEvents).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/events/created", eDelivery.GetEventsFromAuthor).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/subscribers", uDelivery.GetSubscribers).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/subscribes", uDelivery.GetSubscribes).Methods("GET")
	getUserHandlerFunc := mws.Auth(mws.GetVars(http.HandlerFunc(uDelivery.GetUser)))
	r.Handle("", getUserHandlerFunc).Methods("GET")
	r.Handle("/info", useMiddlewares(r, "/info", uDelivery.UpdateUserInfo, mws.GetVars, mws.Auth)).Methods("POST")
	r.Handle("/password", useMiddlewares(r, "/password", uDelivery.UpdateUserPassword, mws.GetVars, mws.Auth)).Methods("POST")
}

func EventHTTPEndpoints(r *mux.Router, delivery *eventHttp.Delivery, mws *middleware.Middlewares) {
	r.HandleFunc("", delivery.GetEventsFromAuthor).Queries("authorid", "{authorid:[0-9]+}").Methods("GET")
	//TODO: Попросить фронт заменить "query" на "title", ибо понятно, почему.
	r.HandleFunc("", delivery.GetEvents).Queries("query", "{query}", "category", "{category}", "tags", "{tags}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Queries("query", "{query}", "category", "{category}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Queries("query", "{query}", "tags", "{tags}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Queries("query", "{query}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Queries("category", "{category}", "tags", "{tags}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Queries("category", "{category}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Queries("tags", "{tags}").Methods("GET")
	r.HandleFunc("", delivery.GetEvents).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}", delivery.GetEventById).Methods("GET")
	updateEventHandlerFunc := mws.Auth(mws.GetVars(http.HandlerFunc(delivery.UpdateEvent)))
	r.Handle("/{id:[0-9]+}", updateEventHandlerFunc).Methods("POST")
	deleteEventHandlerFunc := mws.Auth(mws.GetVars(http.HandlerFunc(delivery.DeleteEvent)))
	r.Handle("/{id:[0-9]+}", deleteEventHandlerFunc).Methods("DELETE")
	createEventHandlerFunc := mws.Auth(mws.GetVars(http.HandlerFunc(delivery.CreateEvent)))
	r.Handle("", createEventHandlerFunc).Methods("POST")
}
