package mock

import (
	"github.com/feedhenry/negotiator/deploy"
	bc "github.com/openshift/origin/pkg/build/api"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
)

type ClientFactory struct {
	PassClient *PassClient
}

func (cf ClientFactory) DefaultDeployClient(host, token string) (deploy.Client, error) {
	if cf.PassClient != nil {
		return cf.PassClient, nil
	}
	return NewPassClient(), nil
}

func NewPassClient() *PassClient {
	return &PassClient{
		Called:  make(map[string]int),
		Asserts: make(map[string]func(interface{}) error),
		Returns: make(map[string]interface{}),
		Error:   make(map[string]error),
	}
}

type PassClient struct {
	Called  map[string]int
	Asserts map[string]func(interface{}) error
	Returns map[string]interface{}
	Error   map[string]error
}

func (pc PassClient) GetDeploymentConfigByName(ns, deploymentName string) (*dcapi.DeploymentConfig, error) {
	pc.Called["GetDeploymentConfigByName"]++
	if e, ok := pc.Error["GetDeploymentConfigByName"]; ok {
		return nil, e
	}
	var ret *dcapi.DeploymentConfig
	if pc.Returns["GetDeploymentConfigByName"] != nil {
		ret = pc.Returns["GetDeploymentConfigByName"].(*dcapi.DeploymentConfig)
	}
	if assert, ok := pc.Asserts["GetDeploymentConfigByName"]; ok {
		return ret, assert(ret)
	}
	return ret, nil

}

func (pc PassClient) FindDeploymentConfigsByLabel(ns string, searchLabels map[string]string) ([]dcapi.DeploymentConfig, error) {
	pc.Called["FindDeploymentConfigsByLabel"]++
	if e, ok := pc.Error["FindDeploymentConfigsByLabel"]; ok {
		return nil, e
	}
	var ret []dcapi.DeploymentConfig
	if pc.Returns["FindDeploymentConfigsByLabel"] != nil {
		ret = pc.Returns["FindDeploymentConfigsByLabel"].([]dcapi.DeploymentConfig)
	}
	if assert, ok := pc.Asserts["FindDeploymentConfigsByLabel"]; ok {
		return ret, assert(ret)
	}
	return ret, nil
}

func (pc PassClient) FindBuildConfigByLabel(ns string, searchLabels map[string]string) (*bc.BuildConfig, error) {
	pc.Called["FindBuildConfigByLabel"]++
	if e, ok := pc.Error["FindBuildConfigByLabel"]; ok {
		return nil, e
	}
	var ret *bc.BuildConfig
	if pc.Returns["FindBuildConfigByLabel"] != nil {
		ret = pc.Returns["FindBuildConfigByLabel"].(*bc.BuildConfig)
	}
	if assert, ok := pc.Asserts["FindBuildConfigByLabel"]; ok {
		return ret, assert(ret)
	}
	return ret, nil
}

func (pc PassClient) CreateServiceInNamespace(ns string, svc *api.Service) (*api.Service, error) {
	pc.Called["CreateServiceInNamespace"]++
	if e, ok := pc.Error["CreateServiceInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["CreateServiceInNamespace"]; ok {
		return svc, assert(svc)
	}
	return svc, nil
}

func (pc PassClient) CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	pc.Called["CreateRouteInNamespace"]++
	if e, ok := pc.Error["CreateRouteInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["CreateRouteInNamespace"]; ok {
		return r, assert(r)
	}
	return r, nil
}

func (pc PassClient) UpdateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	pc.Called["UpdateRouteInNamespace"]++
	if e, ok := pc.Error["UpdateRouteInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["UpdateRouteInNamespace"]; ok {
		return r, assert(r)
	}
	return r, nil
}

func (pc PassClient) CreateImageStream(ns string, i *ioapi.ImageStream) (*ioapi.ImageStream, error) {
	pc.Called["CreateImageStream"]++
	if e, ok := pc.Error["CreateImageStream"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["CreateImageStream"]; ok {
		return i, assert(i)
	}
	return i, nil
}
func (pc PassClient) CreateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error) {
	pc.Called["CreateBuildConfigInNamespace"]++
	if e, ok := pc.Error["CreateBuildConfigInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["CreateBuildConfigInNamespace"]; ok {
		return b, assert(b)
	}
	return b, nil
}

func (pc PassClient) UpdateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error) {
	pc.Called["UpdateBuildConfigInNamespace"]++
	if e, ok := pc.Error["UpdateBuildConfigInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["UpdateBuildConfigInNamespace"]; ok {
		return b, assert(b)
	}
	return b, nil
}

func (pc PassClient) CreateDeployConfigInNamespace(namespace string, d *dcapi.DeploymentConfig) (*dcapi.DeploymentConfig, error) {
	pc.Called["CreateDeployConfigInNamespace"]++
	if e, ok := pc.Error["CreateDeployConfigInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["CreateDeployConfigInNamespace"]; ok {
		return d, assert(d)
	}
	return d, nil
}

func (pc PassClient) UpdateDeployConfigInNamespace(namespace string, d *dcapi.DeploymentConfig) (*dcapi.DeploymentConfig, error) {
	pc.Called["UpdateDeployConfigInNamespace"]++
	if e, ok := pc.Error["UpdateDeployConfigInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["UpdateDeployConfigInNamespace"]; ok {
		return d, assert(d)
	}
	return d, nil
}

func (pc PassClient) CreateSecretInNamespace(namespace string, s *api.Secret) (*api.Secret, error) {
	pc.Called["CreateSecretInNamespace"]++
	if e, ok := pc.Error["CreateSecretInNamespace"]; ok {
		return nil, e
	}
	if assert, ok := pc.Asserts["CreateSecretInNamespace"]; ok {
		return s, assert(s)
	}
	return s, nil
}

func (pc PassClient) InstantiateBuild(ns, buildName string) (*bc.Build, error) {
	pc.Called["InstantiateBuild"]++
	if e, ok := pc.Error["InstantiateBuild"]; ok {
		return nil, e
	}
	var ret *bc.Build
	if pc.Returns["InstantiateBuild"] != nil {
		ret = pc.Returns["InstantiateBuild"].(*bc.Build)
	}
	if assert, ok := pc.Asserts["InstantiateBuild"]; ok {
		return ret, assert(ret)
	}
	return ret, nil
}

func (pc PassClient) DeployLogURL(ns, dc string) string {
	pc.Called["DeployLogURL"]++
	var ret string
	if pc.Returns["DeployLogURL"] != nil {
		ret = pc.Returns["DeployLogURL"].(string)
	}
	return ret
}
func (pc PassClient) BuildConfigLogURL(ns, dc string) string {
	pc.Called["BuildConfigLogURL"]++
	var ret string
	if pc.Returns["BuildConfigLogURL"] != nil {
		ret = pc.Returns["BuildConfigLogURL"].(string)
	}
	return ret
}
func (pc PassClient) BuildURL(ns, bc, id string) string {
	pc.Called["BuildURL"]++
	var ret string
	if pc.Returns["BuildURL"] != nil {
		ret = pc.Returns["BuildURL"].(string)
	}
	return ret
}

func (pc PassClient) CalledTimes(f string) int {
	return pc.Called[f]
}
