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
	PojectGUID    string   `json:"pojectGuid"`
	RepoURL       *string  `json:"repoUrl,omitempty"`
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
	if missing, err := payload.validate(); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		encoder.Encode(missing)
		return
	}
	osServiceName := d.objectNamer.UniqueName("service", payload.GUID)
	osCloudAppName := d.objectNamer.ConsistentName("cloud", payload.GUID)
	namespace := d.namer.Namespace(payload.Namespace)
	labels := map[string]string{
		"rmmap/guid":    payload.GUID,
		"rhmap/project": payload.PojectGUID,
		"rhmap/domain":  payload.Domain,
	}
	if _, err := d.paasService.CreateService(namespace, osServiceName, osCloudAppName, "rhmap cloud app", 8001, labels); err != nil {
		d.handleDeployError(err, "failed to create service ", res)
		return
	}
	if err := d.paasService.CreateRoute(namespace, osServiceName, osCloudAppName, "", labels); err != nil {
		d.handleDeployError(err, "failed to create route ", res)
		return
	}
	if err := d.paasService.CreateImageStream(namespace, osCloudAppName, labels); err != nil {
		d.handleDeployError(err, "failed to create imagestream", res)
		return
	}

	//create secrets

	//create build config

	//create deployment config
}

func (d DeployHandler) handleDeployError(err error, msg string, rw http.ResponseWriter) {
	d.logger.Error(msg, fmt.Sprintf("%+v", err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg))
}
