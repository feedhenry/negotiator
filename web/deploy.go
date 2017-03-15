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
	"github.com/feedhenry/negotiator/pkg/log"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// DeployClientFactory defines how we want to get our OSCP and Kubernetes client
type DeployClientFactory interface {
	DefaultDeployClient(host, token string) (deploy.Client, error)
}

// Deploy handles deploying to OpenShift
type Deploy struct {
	logger        log.Logger
	deploy        *deploy.Controller
	clientFactory DeployClientFactory
}

// NewDeployHandler creates a cloudApp controller.
func NewDeployHandler(logger log.Logger, deployController *deploy.Controller, clientFactory DeployClientFactory) Deploy {
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
		encoder   = json.NewEncoder(res)
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
		d.handleDeployError(err, "validation failed: ", res)
		return
	}
	client, err := d.clientFactory.DefaultDeployClient(payload.Target.Host, payload.Target.Token)
	if err != nil {
		d.handleDeployErrorWithStatus(err, http.StatusUnauthorized, res)
		return
	}
	complete, err := d.deploy.Template(client, template, nameSpace, payload)
	if err != nil {
		d.handleDeployError(err, "unexpected error deploying template: ", res)
		return
	}
	if err := encoder.Encode(complete); err != nil {
		d.handleDeployError(err, "failed to encode response: ", res)
		return
	}
}

func (d Deploy) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	cause := errors.Cause(err)
	if os.IsNotExist(cause) {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	switch err.(type) {
	case *json.SyntaxError, deploy.ErrInvalid:
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(msg))
		return
	}
	d.logger.Error(fmt.Sprintf(" error deploying. context: %s \n %+v", msg, err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg + err.Error()))
}

func (d Deploy) handleDeployErrorWithStatus(err error, status int, rw http.ResponseWriter) {
	d.logger.Error(fmt.Sprintf(" error deploying \n %+v", err))
	rw.WriteHeader(status)
	rw.Write([]byte(err.Error()))
}
