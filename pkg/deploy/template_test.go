package deploy_test

import (
	"errors"
	"fmt"
	"testing"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/pkg/status"
	bc "github.com/openshift/origin/pkg/build/api"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
)

type mockServiceConfigFactory struct{}

func (msf mockServiceConfigFactory) Factory(service string, config *deploy.Configuration, wait *sync.WaitGroup) deploy.Configurer {
	return mockConfigurer{}
}
func (msf *mockServiceConfigFactory) Publisher(pub deploy.StatusPublisher) {

}

type mockConfigurer struct{}

func (mc mockConfigurer) Configure(client deploy.Client, inprogress *dcapi.DeploymentConfig, ns string) (*dcapi.DeploymentConfig, error) {
	return inprogress, nil
}

func TestDeploy(t *testing.T) {
	cases := []struct {
		TestName          string
		Template          string
		NameSpace         string
		Payload           *deploy.Payload
		Calls             map[string]int
		ClientAsserts     map[string]func(interface{}) error
		DispactchedAssert func(d *deploy.Dispatched) error
		Returns           map[string]interface{}
		ExpectError       bool
	}{
		{
			TestName:    "test deploy cache",
			ExpectError: false,
			Returns: map[string]interface{}{
				"CreateDeployConfigInNamespace": &dcapi.DeploymentConfig{
					ObjectMeta: api.ObjectMeta{
						Name:      "cacheservice",
						Namespace: "test",
					},
				},
			},
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 1,
				"CreateServiceInNamespace":      1,
			},
			DispactchedAssert: func(d *deploy.Dispatched) error {
				if d.InstanceID != "test:cacheservice" {
					return fmt.Errorf("expected InstanceID to match %s but got %s", "test:cacheservice", d.InstanceID)
				}
				if d.Operation != "provision" {
					return fmt.Errorf("expected operation to be provision but got %s ", d.Operation)
				}
				return nil
			},
			ClientAsserts: map[string]func(interface{}) error{
				"CreateDeployConfigInNamespace": func(bc interface{}) error {
					if nil == bc {
						return errors.New("did not expect nil DeploymentConfig")
					}
					dep := bc.(*dcapi.DeploymentConfig)
					if dep.Name != "cacheservice" {
						return errors.New("expected service name to be cacheservice but got " + dep.Name)
					}
					return nil
				},
				"CreateServiceInNamespace": func(s interface{}) error {
					if nil == s {
						return errors.New("did not expect nil Service")
					}
					dep := s.(*api.Service)
					if dep.Name != "cacheservice" {
						return errors.New("expected service name to be cacheservice but got " + dep.Name)
					}
					return nil
				},
			},
			Template:  "cache-redis",
			NameSpace: "test",
			Payload: &deploy.Payload{
				Target: &deploy.Target{
					Host:  "http://test.com",
					Token: "test",
				},
				ServiceName:  "cacheservice",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
				},
			},
		},
		{
			TestName: "test deploy cloudapp",
			Returns: map[string]interface{}{
				"InstantiateBuild": &bc.Build{ObjectMeta: api.ObjectMeta{Name: "test"}},
				"CreateDeployConfigInNamespace": &dcapi.DeploymentConfig{
					ObjectMeta: api.ObjectMeta{
						Name:      "cloudapp",
						Namespace: "test",
					},
				},
			},
			ExpectError: false,
			DispactchedAssert: func(d *deploy.Dispatched) error {
				if d.InstanceID != "test:cloudapp" {
					return fmt.Errorf("expected the instanceID to be %s but got %s ", "test:cloudapp", d.InstanceID)
				}
				if d.Operation != "provision" {
					return fmt.Errorf("expected operation to be provision but got %s ", d.Operation)
				}
				return nil
			},
			Template:  "cloudapp",
			NameSpace: "test",
			Payload: &deploy.Payload{
				Target: &deploy.Target{
					Host:  "http://test.com",
					Token: "test",
				},
				ServiceName:  "cloudapp",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
					{
						Name:  "test2",
						Value: "test2",
					},
				},
				Repo: &deploy.Repo{
					Loc: "http://git.test.com",
					Ref: "master",
				},
			},
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 1,
				"CreateServiceInNamespace":      1,
				"CreateImageStream":             1,
				"CreateBuildConfigInNamespace":  1,
				"CreateRouteInNamespace":        1,
			},
		},
		{
			TestName:    "test missing template",
			ExpectError: true,
			Template:    "idontexist",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				ServiceName: "cacheservice",
			},
		},
		{
			TestName:    "test invalid template payload",
			ExpectError: true,
			Template:    "cache-redis",
			NameSpace:   "test",
			Payload:     &deploy.Payload{},
		},
		{
			TestName:    "test invalid cloudapp payload",
			ExpectError: true,
			Template:    "cloudapp",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				ServiceName: "test",
			},
		},
		{
			TestName: "test redeploy cloudapp",
			Returns: map[string]interface{}{
				"InstantiateBuild":             &bc.Build{ObjectMeta: api.ObjectMeta{Name: "test"}},
				"FindBuildConfigByLabel":       &bc.BuildConfig{ObjectMeta: api.ObjectMeta{Name: "test"}},
				"FindDeploymentConfigsByLabel": []dcapi.DeploymentConfig{{ObjectMeta: api.ObjectMeta{Name: "test"}}},
			},
			DispactchedAssert: func(d *deploy.Dispatched) error {
				return nil
			},
			ExpectError: false,
			Template:    "cloudapp",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				Target: &deploy.Target{
					Host:  "http://test.com",
					Token: "test",
				},
				ServiceName:  "cacheservice",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
					{
						Name:  "test2",
						Value: "test2",
					},
				},
				Repo: &deploy.Repo{
					Loc: "http://git.test.com",
					Ref: "master",
				},
			},
			Calls: map[string]int{
				"UpdateDeployConfigInNamespace": 1,
				"UpdateBuildConfigInNamespace":  1,
				"UpdateRouteInNamespace":        1,
			},
		},
		{
			TestName: "test deploy data",
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 1,
				"CreateServiceInNamespace":      1,
				"CreatePersistentVolumeClaim":   1,
			},
			Template:  "data-mongo",
			NameSpace: "test",
			Returns: map[string]interface{}{
				"CreateDeployConfigInNamespace": &dcapi.DeploymentConfig{
					ObjectMeta: api.ObjectMeta{
						Name:      "data",
						Namespace: "test",
					},
				},
			},
			DispactchedAssert: func(d *deploy.Dispatched) error {
				if d.InstanceID != "test:data" {
					return fmt.Errorf("expected the instanceID to be %s but got %s ", "test:data", d.InstanceID)
				}
				if d.Operation != "provision" {
					return fmt.Errorf("expected operation to be provision but got %s ", d.Operation)
				}
				return nil
			},
			Payload: &deploy.Payload{
				Target: &deploy.Target{
					Host:  "http://test.com",
					Token: "test",
				},
				ServiceName:  "data",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
				},
			},
		},
		{
			TestName: "test deploy data with specified size",
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 1,
				"CreateServiceInNamespace":      1,
				"CreatePersistentVolumeClaim":   1,
			},
			DispactchedAssert: func(d *deploy.Dispatched) error {
				return nil
			},
			ClientAsserts: map[string]func(interface{}) error{
				"CreatePersistentVolumeClaim": func(pvc interface{}) error {
					if nil == pvc {
						return errors.New("did not expect nil PersistentVolumeClaim")
					}
					dep := pvc.(*api.PersistentVolumeClaim)
					if nil == dep {
						return fmt.Errorf("expected a PersistentVolumeClaim but got none")
					}
					storage := dep.Spec.Resources.Requests["storage"]
					st := &storage
					if st.String() != "5Gi" {
						return fmt.Errorf("expected 5Gi but got %s", st.String())
					}
					return nil
				},
			},
			Template:  "data-mongo",
			NameSpace: "test",
			Payload: &deploy.Payload{
				Target: &deploy.Target{
					Host:  "http://test.com",
					Token: "test",
				},
				ServiceName:  "dataservice",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				Options: map[string]interface{}{
					"storage": "5Gi",
				},
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
				},
			},
		},
	}
	tl := openshift.NewTemplateLoaderDecoder("../resources/templates/")
	logger := logrus.StandardLogger()
	lsp := &status.LogStatusPublisher{Logger: logger}
	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			pc := mock.NewPassClient()
			if tc.Returns != nil {
				pc.Returns = tc.Returns
			}
			pc.Asserts = tc.ClientAsserts
			sc := deploy.NewEnvironmentServiceConfigController(&mockServiceConfigFactory{}, logger, nil, tl)
			dc := deploy.New(tl, tl, logrus.StandardLogger(), sc, lsp)
			dispatched, err := dc.Template(pc, tc.Template, tc.NameSpace, tc.Payload)
			if !tc.ExpectError && err != nil {
				fmt.Printf("%+v", err)
				t.Fatal(err)
			}
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got nil")
			}
			if tc.DispactchedAssert != nil {
				if err := tc.DispactchedAssert(dispatched); err != nil {
					t.Errorf("Dispatch assert failed. Did not expect an error but got %s", err.Error())
				}
			}
			for f, n := range tc.Calls {
				if n != pc.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times", f, n)
				}
			}
		})
	}
}
