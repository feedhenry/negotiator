package deploy_test

import (
	"fmt"
	"testing"

	"github.com/feedhenry/negotiator/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
)

type mockStatusPublisher struct {
	Statuses []string
}

func (msp *mockStatusPublisher) Publish(key string, status deploy.ConfigurationStatus) error {
	fmt.Println("calling mockstatus publisher", key, status.Status)
	msp.Statuses = append(msp.Statuses, status.Status)
	fmt.Println("status", msp.Statuses)
	return nil
}
func (msp *mockStatusPublisher) Finish(key string) {}

func TestConfiguringCacheJob(t *testing.T) {
	msp := &mockStatusPublisher{}
	cacheConfig := deploy.CacheConfigure{StatusPublisher: msp}
	client := mock.NewPassClient()
	depConfig := []dcapi.DeploymentConfig{
		{ObjectMeta: api.ObjectMeta{Name: "test"}, Spec: dcapi.DeploymentConfigSpec{
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
		}},
	}

	cases := []struct {
		TestName    string
		ExpectError bool
		Assert      func(d *dcapi.DeploymentConfig) error
		Update      func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig
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
					if env.Name == "FH_REDIS_HOST" && env.Value == "cache" {
						varFound = true
					}
				}
				if !varFound {
					return fmt.Errorf("expected to find FH_REDIS_HOST with value cache")
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
					if env.Name == "FH_REDIS_HOST" && env.Value == "cache" {
						varFound = true
					}
				}
				if !varFound {
					return fmt.Errorf("expected to find FH_REDIS_HOST with value cache")
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			client.Returns["FindDeploymentConfigsByLabel"] = depConfig
			deployment, err := cacheConfig.Configure(client, &depConfig[0], "test")
			if tc.ExpectError && err == nil {
				t.Fatalf(" expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatalf(" did not expect an error but got %s", err.Error())
			}
			if err := tc.Assert(deployment); err != nil {
				t.Fatalf("assert error occurred %s ", err.Error())
			}
		})
	}

}
