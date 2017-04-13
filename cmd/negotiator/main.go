//go:generate goagen swagger -d github.com/feedhenry/negotiator/design -o=$GOPATH/src/github.com/feedhenry/negotiator/design
package main

import (
	"net/http"

	"github.com/go-redis/redis"

	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/config"
	"github.com/feedhenry/negotiator/pkg/deploy"
	pkgos "github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/pkg/status"
	"github.com/feedhenry/negotiator/pkg/web"
)

var logLevel string

func setupLogger() {
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
}

func main() {
	flag.StringVar(&logLevel, "log-level", "info", "use this to set log level: error, info, debug")
	flag.Parse()
	conf := config.Conf{}
	clientFactory := pkgos.ClientFactory{}
	setupLogger()
	logger := logrus.StandardLogger()
	router := web.BuildRouter()
	templates := pkgos.NewTemplateLoaderDecoder(conf.TemplateDir())

	var statusPublisher deploy.StatusPublisher
	var statusRetriever web.StatusRetriever
	//status publisher setup
	{
		logrus.Info("using redis publisher")
		redisOpts := conf.Redis()
		redisClient := redis.NewClient(&redisOpts)
		redisPub := status.New(redisClient)
		logPub := &status.LogStatusPublisher{
			Logger: logger,
		}
		pubRet := status.NewMultiStatus(redisPub, logPub, logger)
		statusRetriever = redisPub // it implments both interfaces
		statusPublisher = pubRet
	}
	// deploy setup
	{
		//not use a log publisher this would be replaced with something that published to redis
		serviceConfigFactory := &deploy.ConfigurationFactory{
			StatusPublisher: statusPublisher,
			TemplateLoader:  templates,
			Logger:          logger,
		}
		serviceConfigController := deploy.NewEnvironmentServiceConfigController(serviceConfigFactory, logger, statusPublisher, templates)
		deployController := deploy.New(templates, templates, logger, serviceConfigController, statusPublisher)
		web.DeployRoute(router, logger, deployController, clientFactory)
	}
	// Routes setup

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
		web.Templates(router, templates, clientFactory)
	}
	// LastOperation setup
	{
		web.LastOperation(router, statusRetriever, logger)
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
