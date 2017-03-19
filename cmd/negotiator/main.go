//go:generate goagen swagger -d github.com/feedhenry/negotiator/design -o=$GOPATH/src/github.com/feedhenry/negotiator/design
package main

import (
	"net/http"

	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/config"
	"github.com/feedhenry/negotiator/pkg/deploy"
	pkgos "github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/pkg/web"
)

var logLevel string

func main() {
	flag.StringVar(&logLevel, "log-level", "info", "use this to set log level: error, info, debug")
	flag.Parse()
	conf := config.Conf{}
	clientFactory := pkgos.ClientFactory{}
	router := web.BuildRouter()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	switch logLevel {
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	default:
		logrus.SetLevel(logrus.ErrorLevel)
	}
	logger := logrus.StandardLogger()
	templates := pkgos.NewTemplateLoaderDecoder(conf.TemplateDir())
	// deploy setup
	{
		//not use a log publisher this would be replaced with something that published to redis
		serviceConfigFactory := &deploy.ConfigurationFactory{
			StatusPublisher: deploy.LogStatusPublisher{Logger: logger},
			TemplateLoader:  templates,
		}
		serviceConfigController := deploy.NewEnvironmentServiceConfigController(serviceConfigFactory, logger, nil, templates)
		deployController := deploy.New(templates, templates, logger, serviceConfigController)
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
	// templates setup
	{
		web.Templates(router, templates)
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
