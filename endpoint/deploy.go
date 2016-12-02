package endpoint

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/feedhenry/negotiator/controller"
	"github.com/feedhenry/negotiator/domain/rhmap"
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

type DeployController interface {
	Run(DeployCloudAppPayload) (interface{}, error)
}

// Deploy handles deploying to OpenShift
type Deploy struct {
	logger           Logger
	deployController controller.Deploy
	namer            Namespacer
	objectNamer      ObjectNamer
}

// DeployCloudAppPayload is the JSON structure sent from the client
type DeployCloudAppPayload struct {
	namer              Namespacer
	objectNamer        ObjectNamer
	Domain             string               `json:"domain"`
	EnvVars            []*controller.EnvVar `json:"envVars,omitempty"`
	Environment        string               `json:"environment"`
	GUID               string               `json:"guid"`
	service            string
	// this are the services you require such as redis
	InfraServices      []string          `json:"infraServices,omitempty"`
	Namespace          string            `json:"namespace"`
	ProjectGUID        string            `json:"projectGuid"`
	RepoURL            string            `json:"repoUrl,omitempty"`
	RepoBranchOrCommit string            `json:"repoBranchOrCommit,omitempty"`
	User               string            `json:"user"`
	Auth               string            `json:"auth"`
	AppTag             string            `json:"appTag"`
	labels             map[string]string `json:"labels"`
}

func (dp *DeployCloudAppPayload) EnvironmentName() string {
	return dp.namer.Namespace(dp.Namespace)
}
func (dp *DeployCloudAppPayload) SetLabels(labels map[string]string) {
	dp.labels = labels
}
func (dp *DeployCloudAppPayload) SetAppTag(tag string) {
	dp.AppTag = tag
}
func (dp *DeployCloudAppPayload) GetAppTag() string {
	return dp.AppTag
}
func (dp *DeployCloudAppPayload) AppName() string {
	return dp.objectNamer.ConsistentName(dp.AppTag, dp.GUID)
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
	return dp.RepoBranchOrCommit
}
func (dp *DeployCloudAppPayload) AddEnvVar(name, value string) {
	dp.EnvVars = append(dp.EnvVars, &controller.EnvVar{Key: name, Value: value})
}
func (dp *DeployCloudAppPayload) GetEnvVars() []*controller.EnvVar {
	return dp.EnvVars
}
func (dp *DeployCloudAppPayload) Labels() map[string]string {
	return dp.labels
}

func (dp *DeployCloudAppPayload) validate() ([]string, error) {
	// TODO add validation here
	// TODO add missing fields to array
	return []string{}, nil
}

// NewDeployHandler creates a cloudApp controller.
func NewDeployHandler(logger Logger, deployController controller.Deploy, namer Namespacer) Deploy {
	return Deploy{
		logger:           logger,
		deployController: deployController,
		namer:            namer,
		objectNamer:      rhmap.Service{},
	}
}

// Deploy sends the generated templates to the OpenShift PaaS
func (d Deploy) Deploy(res http.ResponseWriter, req *http.Request) {
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

	if missing, err := payload.validate(); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		encoder.Encode(missing)
		return
	}
	payload.SetLabels(map[string]string{
		"rmmap/guid":    payload.CloudAppGUID(),
		"rhmap/project": payload.Project(),
		"rhmap/domain":  payload.DomainName(),
	})
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

func (d Deploy) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	d.logger.Error(fmt.Sprintf("%s \n %+v", msg, err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg))
}
