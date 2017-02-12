package mock

import (
	bc "github.com/openshift/origin/pkg/build/api"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
)

type PassClient struct {
}

func (pc PassClient) ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error) {
	return nil, nil

}
func (pc PassClient) CreateServiceInNamespace(ns string, svc *api.Service) (*api.Service, error) {
	return nil, nil
}
func (pc PassClient) CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	return nil, nil
}
func (pc PassClient) CreateImageStream(ns string, i *ioapi.ImageStream) (*ioapi.ImageStream, error) {
	return nil, nil
}
func (pc PassClient) CreateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error) {
	return nil, nil
}
func (pc PassClient) CreateDeployConfigInNamespace(namespace string, d *dcapi.DeploymentConfig) (*dcapi.DeploymentConfig, error) {
	return nil, nil
}
func (pc PassClient) CreateSecretInNamespace(namespace string, s *api.Secret) (*api.Secret, error) {
	return nil, nil
}

func (pc PassClient)InstantiateBuild(ns, buildName string) (*bc.Build, error){
	return nil, nil
}
