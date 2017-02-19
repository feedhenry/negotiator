package web

/*
package web is responsible for the web layer. It is an out layer and depends on the inner business domains to do its work.
It should not be required from internal business logic
*/

import (
	"encoding/json"
	"net/http"

	"fmt"

	"os"

	"github.com/feedhenry/negotiator/deploy"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// Logger describes a logging interface
type Logger interface {
	Info(args ...interface{})
	Error(args ...interface{})
}

// DeployClientFactory defines how we want to get our OSCP and Kubernetes client
type DeployClientFactory interface {
	DefaultDeployClient(host, token string) (deploy.DeployClient, error)
}

// Deploy handles deploying to OpenShift
type Deploy struct {
	logger        Logger
	deploy        *deploy.Controller
	clientFactory DeployClientFactory
}

// NewDeployHandler creates a cloudApp controller.
func NewDeployHandler(logger Logger, deployController *deploy.Controller, clientFactory DeployClientFactory) Deploy {
	return Deploy{
		logger:        logger,
		deploy:        deployController,
		clientFactory: clientFactory,
	}
}

// Deploy decodes the deploy payload. Pulls together a client which is then used to send generated templates to the OpenShift PaaS
func (d Deploy) Deploy(res http.ResponseWriter, req *http.Request) {
	var (
		decoder   = json.NewDecoder(req.Body)
		params    = mux.Vars(req)
		template  = params["template"]
		nameSpace = params["nameSpace"]
		payload   = &deploy.Payload{}
	)
	if err := decoder.Decode(payload); err != nil {
		d.handleDeployError(err, "failed to decode json "+err.Error(), res)
		return
	}
	if err := payload.Validate(template); err != nil {
		d.handleDeployErrorWithStatus(err, http.StatusBadRequest, res)
		return
	}
	client, err := d.clientFactory.DefaultDeployClient(payload.Target.Host, payload.Target.Token)
	if err != nil {
		d.handleDeployErrorWithStatus(err, http.StatusUnauthorized, res)
		return
	}
	if err := d.deploy.Template(client, template, nameSpace, payload); err != nil {
		d.handleDeployError(err, "unexpected error deploying template", res)
		return
	}
	res.WriteHeader(201)
}

func (d Deploy) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	cause := errors.Cause(err)
	if os.IsNotExist(cause) {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	switch err.(type) {
	case *json.SyntaxError:
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(msg))
		return
	}
	d.logger.Error(fmt.Sprintf(" error deploying. context: %s \n %+v", msg, err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg))
}

func (d Deploy) handleDeployErrorWithStatus(err error, status int, rw http.ResponseWriter) {
	d.logger.Error(fmt.Sprintf(" error deploying \n %+v", err))
	rw.WriteHeader(status)
	rw.Write([]byte(err.Error()))
}
