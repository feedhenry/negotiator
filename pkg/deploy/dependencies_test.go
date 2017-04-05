package deploy_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
)

func TestDeployWithDependencies(t *testing.T) {
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
			TestName:    "test deploy ups with dependency on mysql",
			ExpectError: false,
			Returns: map[string]interface{}{
				"CreateDeployConfigInNamespace": &dcapi.DeploymentConfig{
					ObjectMeta: api.ObjectMeta{
						Name:      "push-ups",
						Namespace: "test",
					},
				},
				"GetDeployLogs": "success",
			},
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 2,
				"CreateServiceInNamespace":      2,
			},
			DispactchedAssert: func(d *deploy.Dispatched) error {
				if d.InstanceID != "test:push-ups" {
					return fmt.Errorf("expected InstanceID to match %s but got %s", "test:push-ups", d.InstanceID)
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
					if dep.Name != "push-ups" && dep.Name != "data-mysql" {
						return errors.New("expected deploy-config name to be push-ups but got " + dep.Name)
					}
					return nil
				},
				"CreateServiceInNamespace": func(s interface{}) error {
					if nil == s {
						return errors.New("did not expect nil Service")
					}
					dep := s.(*api.Service)
					if dep.Name != "push-ups" && dep.Name != "data-mysql" {
						return errors.New("expected service name to be push-ups or data-mysql but got " + dep.Name)
					}
					return nil
				},
			},
			Template:  "push-ups",
			NameSpace: "test",
			Payload: &deploy.Payload{
				Target: &deploy.Target{
					Host:  "http://test.com",
					Token: "test",
				},
				ServiceName:  "push-ups",
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
	}
	tl := openshift.NewTemplateLoaderDecoder("../resources/templates/")
	logger := logrus.StandardLogger()
	lsp := deploy.LogStatusPublisher{Logger: logger}
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
					t.Errorf("Expected %s to be called %d times, but it was called %d times", f, n, pc.CalledTimes(f))
				}
			}
		})
	}
}
