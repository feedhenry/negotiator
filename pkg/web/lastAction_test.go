package web_test

import (
	"net/http"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/web"
)

type mockStatusRetriever struct {
	status *deploy.ConfigurationStatus
	err    error
}

func (msr mockStatusRetriever) Get(key string) (*deploy.ConfigurationStatus, error) {
	return msr.status, msr.err
}

func setUpLastActionHandler(statusRetriever web.StatusRetriever) http.Handler {
	router := web.BuildRouter()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logger := logrus.StandardLogger()
	web.LastAction(router, statusRetriever, logger)
	return web.BuildHTTPHandler(router)
}

//GET /v2/service_instances/:instance_id/last_operation
func TestGetLastAction(t *testing.T) {

}
