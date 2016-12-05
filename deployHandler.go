package main

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/feedhenry/negotiator/controller"
	"github.com/feedhenry/negotiator/domain/rhmap"
)

// Namespacer is something that know what the correct namespace is
type Namespacer interface {
	Namespace(override string) string
}

// ObjectNamer knows how to name Objects for the OSCP and Kubernetes API
type ObjectNamer interface {
	UniqueName(prefix, guid string) string
	ConsistentName(prefix, guid string) string
}

type DeployController interface {
	Run(controller.DeployCmd) (interface{}, error)
}

// DeployHandler handles deploying to OpenShift
type DeployHandler struct {
	logger           Logger
	deployController controller.Deploy
	namer            Namespacer
	objectNamer      ObjectNamer
}

// EnvVar defines an environment variable
type EnvVar struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// DeployCloudAppPayload is the JSON structure sent from the client
type DeployCloudAppPayload struct {
	namer       Namespacer
	objectNamer ObjectNamer
	Domain      string    `json:"domain"`
	EnvVars     []*EnvVar `json:"envVars,omitempty"`
	Environment string    `json:"environment"`
	GUID        string    `json:"guid"`
	// this are the services you require such as redis
	InfraServices []string `json:"infraServices,omitempty"`
	Namespace     string   `json:"namespace"`
	ProjectGUID   string   `json:"projectGuid"`
	RepoURL       string   `json:"repoUrl,omitempty"`
	RepoBranch    string   `json:"repoBranch,omitempty"`
	User          string   `json:"user"`
	Auth          string   `json:"auth"`
}

func (dp *DeployCloudAppPayload) EnvironmentName() string {
	return dp.namer.Namespace(dp.Namespace)
}
func (dp *DeployCloudAppPayload) CloudAppName() string {
	return dp.objectNamer.ConsistentName("cloud", dp.GUID)
}
func (dp *DeployCloudAppPayload) BuildConfigName() string {
	return dp.objectNamer.ConsistentName("buildconfig", dp.GUID)
}
func (dp *DeployCloudAppPayload) ServiceName() string {
	return dp.objectNamer.UniqueName("service", dp.GUID)
}
func (dp *DeployCloudAppPayload) CloudAppGUID() string {
	return dp.GUID
}
func (dp *DeployCloudAppPayload) Project() string {
	return dp.ProjectGUID
}
func (dp *DeployCloudAppPayload) DomainName() string {
	return dp.Domain
}
func (dp *DeployCloudAppPayload) UserName() string {
	return dp.User
}
func (dp *DeployCloudAppPayload) Authentication() string {
	return dp.Auth
}
func (dp *DeployCloudAppPayload) SourceLoc() string {
	return dp.RepoURL
}
func (dp *DeployCloudAppPayload) SourceBranch() string {
	return dp.RepoBranch
}

func (dc *DeployCloudAppPayload) validate() ([]string, error) {
	// add validation here
	// add missing fields to array
	return []string{}, nil
}

// NewDeployHandler creates a cloudApp controller.
func NewDeployHandler(logger Logger, deployController controller.Deploy, namer Namespacer) DeployHandler {
	return DeployHandler{
		logger:           logger,
		deployController: deployController,
		namer:            namer,
		objectNamer:      rhmap.Service{},
	}
}

// Deploy sends the generated templates to the OpenShift PaaS
func (d DeployHandler) Deploy(res http.ResponseWriter, req *http.Request) {
	d.logger.Info("running deploy")
	var (
		encoder = json.NewEncoder(res)
		decoder = json.NewDecoder(req.Body)
	)

	payload := &DeployCloudAppPayload{namer: d.namer, objectNamer: d.objectNamer}
	if err := decoder.Decode(payload); err != nil {
		d.logger.Error("failed to decode json ", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("failed to decode json"))
		return
	}

	d.logger.Info("Succesfully decoded the payload")

	if missing, err := payload.validate(); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		encoder.Encode(missing)
		return
	}
	response, err := d.deployController.Run(payload)
	if err != nil {
		d.logger.Error(fmt.Sprintf("failed to deploy:\n %+v", err))
		res.WriteHeader(http.StatusInternalServerError) //make more specific
		res.Write([]byte(err.Error()))
		return
	}
	if err := encoder.Encode(response); err != nil {
		d.logger.Error("failed to encode response %s ", err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	//create deployment config
}

func (d DeployHandler) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	d.logger.Error(msg)
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg))
}
