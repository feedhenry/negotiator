package web

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/feedhenry/negotiator/deploy"
	"github.com/gorilla/mux"
)

// Logger describes a logging interface
type Logger interface {
	Info(args ...interface{})
	Error(args ...interface{})
}

// Namespacer is something that know what the correct namespace is
type Namespacer interface {
	Namespace(override string) string
}

// ObjectNamer knows how to name Objects for the OSCP and Kubernetes API
type ObjectNamer interface {
	UniqueName(prefix, guid string) string
	ConsistentName(prefix, guid string) string
}

// Deploy handles deploying to OpenShift
type Deploy struct {
	logger           Logger
	deployController *deploy.Controller
}

// NewDeployHandler creates a cloudApp controller.
func NewDeployHandler(logger Logger, deployController *deploy.Controller) Deploy {
	return Deploy{
		logger:           logger,
		deployController: deployController,
	}
}

// Deploy sends the generated templates to the OpenShift PaaS
func (d Deploy) Deploy(res http.ResponseWriter, req *http.Request) {
	d.logger.Info("running deploy")
	var (
		decoder = json.NewDecoder(req.Body)
	)
	params := mux.Vars(req)
	template := params["template"]
	nameSpace := params["nameSpace"]

	payload := &deploy.Payload{}
	if err := decoder.Decode(payload); err != nil {
		d.logger.Error("failed to decode json ", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("failed to decode json"))
		return
	}

	if err := payload.Validate(template); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}

	err := d.deployController.Template(template, nameSpace, payload)
	if err != nil {
		d.logger.Error(fmt.Sprintf("failed to deploy:\n %+v", err))
		res.WriteHeader(http.StatusInternalServerError) //make more specific
		res.Write([]byte(err.Error()))
		return
	}
	res.WriteHeader(201)
}

func (d Deploy) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	d.logger.Error(fmt.Sprintf("%s \n %+v", msg, err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg))
}
