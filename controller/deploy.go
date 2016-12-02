package controller

import (
	"k8s.io/kubernetes/pkg/api"
)

// PaaSService defines what the handler expects from a service interacting with the PAAS
type PaaSService interface {
	CreateService(namespace, serviceName, selector, description string, port int32, labels map[string]string) (*api.Service, error)
	CreateRoute(namespace, serviceToBindTo, appName, optionalHost string, labels map[string]string) error
	CreateImageStream(namespace, name string, labels map[string]string) error
	CreateSecret(namespace, name string) error
	CreateBuildConfig(namespace, name, selector, description, gitURL, gitBranch string, labels map[string]string) error
	CreateDeploymentConfig(namespace, name string) error
}

type Deploy struct {
	paaSService PaaSService
}

func NewDeployController(paaSService PaaSService) Deploy {
	return Deploy{
		paaSService: paaSService,
	}
}

// DeployCmd encapsulates what data is required to do a deploy
type DeployCmd interface {
	EnvironmentName() string
	CloudAppName() string
	BuildConfigName() string
	ServiceName() string
	CloudAppGUID() string
	Project() string
	DomainName() string
	UserName() string
	Authentication() string
	SourceLoc() string
	SourceBranch() string
}

type DeployResponse struct {
	Status string `json:"status"`
	LogURL string `json:"logURL"`
}

func (d Deploy) Run(dc DeployCmd) (interface{}, error) {
	exists, err := d.buildConfigExists(dc.BuildConfigName())
	if err != nil {
		return nil, err
	}
	if exists {
		return d.update(dc)
	}
	return d.create(dc)
}

func (d Deploy) buildConfigExists(name string) (bool, error) {
	return false, nil
}

func (d Deploy) create(dc DeployCmd) (*DeployResponse, error) {
	labels := map[string]string{
		"rmmap/guid":    dc.CloudAppGUID(),
		"rhmap/project": dc.Project(),
		"rhmap/domain":  dc.DomainName(),
	}
	if _, err := d.paaSService.CreateService(dc.EnvironmentName(), dc.ServiceName(), dc.CloudAppName(), "rhmap cloud app", 8001, labels); err != nil {
		return nil, err
	}
	if err := d.paaSService.CreateRoute(dc.EnvironmentName(), dc.ServiceName(), dc.CloudAppName(), "", labels); err != nil {
		return nil, err
	}
	if err := d.paaSService.CreateImageStream(dc.EnvironmentName(), dc.CloudAppName(), labels); err != nil {
		return nil, err
	}
	//create secrets

	//create build config
	if err := d.paaSService.CreateBuildConfig(dc.EnvironmentName(), dc.BuildConfigName(), dc.CloudAppName(), "rhmap cloud app", dc.SourceLoc(), dc.SourceBranch(), labels); err != nil {
		return nil, err
	}
	return &DeployResponse{Status: "inprogress", LogURL: "http://mybuildlogurl.com/url"}, nil
}

func (d Deploy) update(dc DeployCmd) (DeployResponse, error) {
	//update logic
	return DeployResponse{}, nil
}
