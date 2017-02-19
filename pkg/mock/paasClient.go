package mock

import (
	"github.com/feedhenry/negotiator/deploy"
	bc "github.com/openshift/origin/pkg/build/api"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
)

type ClientFactory struct{}

func (ClientFactory) DefaultDeployClient(host, token string) (deploy.DeployClient, error) {
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

func (pc PassClient) ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error) {
	pc.Called["ListBuildConfigs"]++
	if e, ok := pc.Error["ListBuildConfigs"]; ok {
		return nil, e
	}
	var ret *bcv1.BuildConfigList
	if pc.Returns["ListBuildConfigs"] != nil {
		ret = pc.Returns["ListBuildConfigs"].(*bcv1.BuildConfigList)
	}
	if assert, ok := pc.Asserts["ListBuildConfigs"]; ok {
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

func (pc PassClient) CalledTimes(f string) int {
	return pc.Called[f]
}
