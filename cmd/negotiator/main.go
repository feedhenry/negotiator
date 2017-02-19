package main

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/config"
	"github.com/feedhenry/negotiator/deploy"
	pkgos "github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/web"
)

func main() {
	conf := config.Conf{}
	clientFactory := pkgos.ClientFactory{}
	router := web.BuildRouter()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logger := logrus.StandardLogger()

	// deploy setup
	{
		templates := pkgos.NewTemplateLoaderDecoder(conf.TemplateDir())
		deployController := deploy.New(templates, templates)
		web.DeployRoute(router, logger, deployController, clientFactory)
	}
	// system setup
	{
		web.SysRoute(router)
	}
	// metrics setup
	{
		web.Metrics(router)
	}
	//http handler
	{
		port := ":3000"
		logrus.Info("starting negotiator on  port " + port)
		httpHandler := web.BuildHTTPHandler(router)
		if err := http.ListenAndServe(port, httpHandler); err != nil {
			logrus.Fatal(err)
		}
	}

}
