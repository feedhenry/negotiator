package web

import (
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/feedhenry/negotiator/deploy"
	"github.com/gorilla/mux"
)

func BuildRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	sysHandler := buildSysHandler()
	r.HandleFunc("/sys/info/health", sysHandler.Health)
	r.HandleFunc("/sys/info/ping", sysHandler.Ping)
	return r
}

func BuildHTTPHandler(r *mux.Router) http.Handler {
	//recovery middleware for any panics in the handlers
	recovery := negroni.NewRecovery()
	recovery.PrintStack = false
	//add middleware for all routes
	n := negroni.New(recovery)
	// set up sys routes
	n.UseHandler(r)
	return n
}

// DeployRoute sets up the deploy route
func DeployRoute(r *mux.Router, logger Logger, controller *deploy.Controller, clientFactory DeployClientFactory) {
	deployHandler := NewDeployHandler(logger, controller, clientFactory)
	r.HandleFunc("/deploy/{nameSpace}/{template}", deployHandler.Deploy).Methods("POST")
}
