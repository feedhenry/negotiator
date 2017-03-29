package deploy_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
)

func TestCacheConfigurationJob(t *testing.T) {
	tl := openshift.NewTemplateLoaderDecoder("../openshift/templates/")
	msp := &mockStatusPublisher{}
	logger := logrus.StandardLogger()
	factory := deploy.ConfigurationFactory{
		StatusPublisher: msp,
		Logger:          logger,
		TemplateLoader:  tl,
	}
	cacheConfig := factory.Factory("cache-redis", &deploy.Configuration{InstanceID: "test", Action: "provision"}, &sync.WaitGroup{})
	serviceDepConfig := []dcapi.DeploymentConfig{
		{
			ObjectMeta: api.ObjectMeta{
				Name: "test",
			},
			Spec: dcapi.DeploymentConfigSpec{
				Template: &api.PodTemplateSpec{
					Spec: api.PodSpec{
						Containers: []api.Container{{
							Name: "",
							Env: []api.EnvVar{{
								Name:  "test",
								Value: "test",
							}, {
								Name:  "FH_REDIS_HOST",
								Value: "",
							}},
						}},
					},
				},
			},
		},
	}

	cases := []struct {
		TestName    string
		ExpectError bool
		Assert      func(d *dcapi.DeploymentConfig) error
		Update      func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig
		Calls       map[string]int
	}{
		{
			TestName:    "test cache updated the correct env var",
			ExpectError: false,
			Update: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				env := []api.EnvVar{
					api.EnvVar{
						Name:  "test",
						Value: "test",
					},
					api.EnvVar{
						Name:  "FH_REDIS_HOST",
						Value: "",
					},
				}
				d.Spec.Template.Spec.Containers[0].Env = env
				return d
			},
			Assert: func(d *dcapi.DeploymentConfig) error {
				if nil == d {
					return fmt.Errorf("expected a DeploymentConfig but got none")
				}
				varFound := false
				for _, env := range d.Spec.Template.Spec.Containers[0].Env {
					if env.Name == "FH_REDIS_HOST" && env.Value == "data-cache" {
						varFound = true
					}
				}
				if !varFound {
					return fmt.Errorf("expected to find FH_REDIS_HOST with value data-cache")
				}
				return nil
			},
		},
		{
			TestName:    "test cache update already set env var if it is wrong",
			ExpectError: false,
			Update: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				env := []api.EnvVar{
					api.EnvVar{
						Name:  "test",
						Value: "test",
					},
					api.EnvVar{
						Name:  "FH_REDIS_HOST",
						Value: "cached",
					},
				}
				d.Spec.Template.Spec.Containers[0].Env = env
				return d
			},
			Assert: func(d *dcapi.DeploymentConfig) error {
				if nil == d {
					return fmt.Errorf("expected a DeploymentConfig but got none")
				}
				varFound := false
				for _, env := range d.Spec.Template.Spec.Containers[0].Env {
					if env.Name == "FH_REDIS_HOST" && env.Value == "data-cache" {
						varFound = true
					}
				}
				if !varFound {
					return fmt.Errorf("expected to find FH_REDIS_HOST with value data-cache")
				}
				return nil
			},
			Calls: map[string]int{},
		},
	}
	//run our test cases
	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			client := mock.NewPassClient()
			client.Returns["FindDeploymentConfigsByLabel"] = serviceDepConfig
			deployment, err := cacheConfig.Configure(client, &serviceDepConfig[0], "test")
			if tc.ExpectError && err == nil {
				t.Fatalf(" expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatalf(" did not expect an error but got %s", err.Error())
			}
			if err := tc.Assert(deployment); err != nil {
				t.Fatalf("assert error occurred %s ", err.Error())
			}
			for f, n := range tc.Calls {
				if n != client.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times", f, n)
				}
			}
		})
	}
}
