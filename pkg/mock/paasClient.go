package mock

import (
	"github.com/feedhenry/negotiator/pkg/deploy"
	bc "github.com/openshift/origin/pkg/build/api"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/watch"
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

func (pc PassClient) CreatePod(ns string, p *api.Pod) (*api.Pod, error) {
	pc.Called["CreateJobToWatch"]++
	if e, ok := pc.Error["CreateJobToWatch"]; ok {
		return nil, e
	}
	ret := p
	if pc.Returns["CreateJobToWatch"] != nil {
		ret = pc.Returns["CreateJobToWatch"].(*api.Pod)
	}
	if assert, ok := pc.Asserts["CreateJobToWatch"]; ok {
		return ret, assert(ret)
	}
	return ret, nil

}

func (pc PassClient) CreateJobToWatch(j *batch.Job, ns string) (watch.Interface, error) {
	pc.Called["CreateJobToWatch"]++
	if e, ok := pc.Error["CreateJobToWatch"]; ok {
		return nil, e
	}
	return watch.NewEmptyWatch(), nil
}

func (pc PassClient) FindRouteByName(ns, name string) (*roapi.Route, error) {
	pc.Called["FindRouteByName"]++
	if e, ok := pc.Error["FindRouteByName"]; ok {
		return nil, e
	}
	var ret = &roapi.Route{}
	if pc.Returns["FindRouteByName"] != nil {
		ret = pc.Returns["FindRouteByName"].(*roapi.Route)
	}
	return ret, nil
}

func (pc PassClient) FindConfigMapByName(ns, name string) (*api.ConfigMap, error) {
	pc.Called["FindConfigMapByName"]++
	if e, ok := pc.Error["FindConfigMapByName"]; ok {
		return nil, e
	}
	var ret = &api.ConfigMap{}
	if pc.Returns["FindConfigMapByName"] != nil {
		ret = pc.Returns["FindConfigMapByName"].(*api.ConfigMap)
	}
	return ret, nil
}

func (pc PassClient) CreateConfigMap(ns string, cm *api.ConfigMap) (*api.ConfigMap, error) {
	pc.Called["CreateConfigMap"]++
	if e, ok := pc.Error["CreateConfigMap"]; ok {
		return nil, e
	}
	var ret = &api.ConfigMap{}
	if pc.Returns["CreateConfigMap"] != nil {
		ret = pc.Returns["CreateConfigMap"].(*api.ConfigMap)
	}
	return ret, nil
}

func (pc PassClient) UpdateConfigMap(ns string, cm *api.ConfigMap) (*api.ConfigMap, error) {
	pc.Called["UpdateConfigMap"]++
	if e, ok := pc.Error["UpdateConfigMap"]; ok {
		return nil, e
	}
	var ret = &api.ConfigMap{}
	if pc.Returns["UpdateConfigMap"] != nil {
		ret = pc.Returns["UpdateConfigMap"].(*api.ConfigMap)
	}
	return ret, nil
}

func (pc PassClient) FindServiceByLabel(ns string, searchLabels map[string]string) ([]api.Service, error) {
	pc.Called["FindServiceByLabel"]++
	if e, ok := pc.Error["FindServiceByLabel"]; ok {
		return nil, e
	}
	var ret = []api.Service{}
	if pc.Returns["FindServiceByLabel"] != nil {
		ret = pc.Returns["FindServiceByLabel"].([]api.Service)
	}
	if assert, ok := pc.Asserts["FindServiceByLabel"]; ok {
		return ret, assert(ret)
	}
	return ret, nil
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

func (pc PassClient) CreatePersistentVolumeClaim(namespace string, claim *api.PersistentVolumeClaim) (*api.PersistentVolumeClaim, error) {
	pc.Called["CreatePersistentVolumeClaim"]++
	if e, ok := pc.Error["CreatePersistentVolumeClaim"]; ok {
		return nil, e
	}
	var ret = claim
	if pc.Returns["CreatePersistentVolumeClaim"] != nil {
		ret = pc.Returns["CreatePersistentVolumeClaim"].(*api.PersistentVolumeClaim)
	}
	if assert, ok := pc.Asserts["CreatePersistentVolumeClaim"]; ok {
		return ret, assert(ret)
	}
	return ret, nil
}

func (pc PassClient) FindJobByName(ns, name string) (*batch.Job, error) {
	pc.Called["FindJobByName"]++
	if e, ok := pc.Error["FindJobByName"]; ok {
		return nil, e
	}
	var ret *batch.Job
	if pc.Returns["FindJobByName"] != nil {
		ret = pc.Returns["FindJobByName"].(*batch.Job)
	}
	if assert, ok := pc.Asserts["FindJobByName"]; ok {
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
	var ret = d
	if pc.Returns["CreateDeployConfigInNamespace"] != nil {
		ret = pc.Returns["CreateDeployConfigInNamespace"].(*dcapi.DeploymentConfig)
	}
	if assert, ok := pc.Asserts["CreateDeployConfigInNamespace"]; ok {
		return ret, assert(d)
	}
	return ret, nil
}

func (pc PassClient) UpdateDeployConfigInNamespace(namespace string, d *dcapi.DeploymentConfig) (*dcapi.DeploymentConfig, error) {
	pc.Called["UpdateDeployConfigInNamespace"]++
	if e, ok := pc.Error["UpdateDeployConfigInNamespace"]; ok {
		return nil, e
	}
	var ret = d
	if pc.Returns["UpdateDeployConfigInNamespace"] != nil {
		ret = pc.Returns["UpdateDeployConfigInNamespace"].(*dcapi.DeploymentConfig)
	}
	if assert, ok := pc.Asserts["UpdateDeployConfigInNamespace"]; ok {
		return d, assert(d)
	}
	return ret, nil
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

func (pc PassClient) GetDeployLogs(ns, deploy string) (string, error) {
	pc.Called["GetDeployLogs"]++
	var ret string
	if pc.Returns["GetDeployLogs"] != nil {
		ret = pc.Returns["GetDeployLogs"].(string)
	}
	return ret, nil
}
