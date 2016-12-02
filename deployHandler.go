package main

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/feedhenry/negotiator/domain/rhmap"
	"k8s.io/kubernetes/pkg/api"
)

// PaaSService defines what the handler expects from a service interacting with the PAAS
type PaaSService interface {
	CreateService(namespace, serviceName, selector, description string, port int32, labels map[string]string) (*api.Service, error)
	CreateRoute(namespace, serviceToBindTo, appName, optionalHost string, labels map[string]string) error
	CreateImageStream(namespace, name string, labels map[string]string) error
	CreateSecret(namespace, name string) error
	CreateBuildConfig(namespace, name, selector, description, gitUrl, gitBranch string, labels map[string]string) error
	CreateDeploymentConfig(namespace, name string) error
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

// DeployHandler handles deploying to OpenShift
type DeployHandler struct {
	logger      Logger
	paasService PaaSService
	namer       Namespacer
	objectNamer ObjectNamer
}

// EnvVar defines an environment variable
type EnvVar struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// DeployCloudAppPayload is the JSON structure sent from the client
type DeployCloudAppPayload struct {
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
}

func (dc *DeployCloudAppPayload) validate() ([]string, error) {
	// add validation here
	// add missing fields to array
	return []string{}, nil
}

// NewDeployHandler creates a cloudApp controller.
func NewDeployHandler(logger Logger, paasService PaaSService, namer Namespacer) DeployHandler {
	return DeployHandler{
		logger:      logger,
		paasService: paasService,
		namer:       namer,
		objectNamer: rhmap.Service{},
	}
}

// Deploy sends the generated templates to the OpenShift PaaS
func (d DeployHandler) Deploy(res http.ResponseWriter, req *http.Request) {
	d.logger.Info("running deploy")
	var (
		encoder = json.NewEncoder(res)
		decoder = json.NewDecoder(req.Body)
	)

	payload := &DeployCloudAppPayload{}
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

	d.logger.Info("Succesfully validated the payload")

	osServiceName := d.objectNamer.UniqueName("service", payload.GUID)
	osCloudAppName := d.objectNamer.ConsistentName("cloud", payload.GUID)
	osBuildConfigName := d.objectNamer.ConsistentName("buildconfig", payload.GUID)
	namespace := d.namer.Namespace(payload.Namespace)
	labels := map[string]string{
		"rmmap/guid":    payload.GUID,
		"rhmap/project": payload.ProjectGUID,
		"rhmap/domain":  payload.Domain,
	}
	if _, err := d.paasService.CreateService(namespace, osServiceName, osCloudAppName, "rhmap cloud app", 8001, labels); err != nil {
		d.logger.Error(err)
		d.handleDeployError(err, "failed to create service ", res)
		return
	}
	d.logger.Info("deployed service")
	if err := d.paasService.CreateRoute(namespace, osServiceName, osCloudAppName, "", labels); err != nil {
		d.logger.Error(err)
		d.handleDeployError(err, "failed to create route ", res)
		return
	}
	d.logger.Info("deployed route")
	if err := d.paasService.CreateImageStream(namespace, osCloudAppName, labels); err != nil {
		d.logger.Error(err)
		d.handleDeployError(err, "failed to create imagestream", res)
		return
	}
	d.logger.Info("deployed image stream")

	//create secrets

	//create build config
	d.logger.Info("deploying build config")
	if err := d.paasService.CreateBuildConfig(namespace, osBuildConfigName, osCloudAppName, "rhmap cloud app", payload.RepoURL, payload.RepoBranch, labels); err != nil {
		d.logger.Error(err)
		d.handleDeployError(err, "failed to deploy build config", res)
		return
	}
	d.logger.Info("deployed build config")

	//create deployment config
}

func (d DeployHandler) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	d.logger.Error(msg, fmt.Sprintf("%+v", err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg))
}
