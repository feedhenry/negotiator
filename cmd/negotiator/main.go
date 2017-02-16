package main

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/feedhenry/negotiator/config"
	"github.com/feedhenry/negotiator/deploy"
	pkgos "github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/web"
	"github.com/gorilla/mux"
)

var conf = config.Conf{}

// Builds the openshift service that encapsulates logic for creating the objects via the pkg/client
// func buildPaasService() deploy.PaaSClient {
// 	//TODO these will need to change for multitenant
// 	host := conf.APIHost()
// 	token := conf.APIToken()
// 	clientConf := pkgos.BuildDefaultConfig(host, token)
// 	client, err := pkgos.ClientFromConfig(clientConf)
// 	if err != nil {
// 		log.Fatalf("err %+v", errors.Wrap(err, "error"))
// 	}
// 	return client
// }

func buildDeployController() *deploy.Controller {
	tl := pkgos.NewTemplateLoader(conf.RepoDir())
	return deploy.New(tl, tl)
}

func buildSysHandler() web.SysHandler {
	return web.SysHandler{}
}

func buildDeployHandler() web.Deploy {
	return web.NewDeployHandler(logrus.StandardLogger(), buildDeployController())
}

func buildHTTPHandler() http.Handler {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	r := mux.NewRouter().StrictSlash(true)
	//recovery middleware for any panics in the handlers
	recovery := negroni.NewRecovery()
	recovery.PrintStack = false
	//add middleware for all routes
	n := negroni.New(recovery)
	// set up sys routes
	sysHandler := buildSysHandler()
	r.HandleFunc("/sys/info/health", sysHandler.Health)
	r.HandleFunc("/sys/info/ping", sysHandler.Ping)
	// set up deploy routes
	deployHandler := buildDeployHandler()
	r.HandleFunc("/deploy/{nameSpace}/{template}", deployHandler.Deploy).Methods("POST")
	n.UseHandler(r)
	return n
}

func main() {
	httpHandler := buildHTTPHandler()
	port := ":3000"
	logrus.Info("starting negotiator on  port " + port)
	if err := http.ListenAndServe(port, httpHandler); err != nil {
		logrus.Fatal(err)
	}
}
