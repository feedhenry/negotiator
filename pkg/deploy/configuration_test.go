package deploy_test

import (
	"fmt"
	"testing"

	"strings"

	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
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
			Calls: map[string]int{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			client := mock.NewPassClient()
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
			for f, n := range tc.Calls {
				if n != client.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times", f, n)
				}
			}
		})
	}

}

func TestDataConfigurationJob(t *testing.T) {
	tl := openshift.NewTemplateLoaderDecoder("../resources/templates/")
	msp := &mockStatusPublisher{}
	dataConfig := deploy.DataConfigure{StatusPublisher: msp, TemplateLoader: tl}
	cases := []struct {
		TestName    string
		ExpectError bool
		Assert      func(d *dcapi.DeploymentConfig) error
		UpdateDC    func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig
		UpdateSVC   func(d *api.Service) *api.Service
		UpdateJob   func(j *batch.Job) *batch.Job
		Calls       map[string]int
	}{
		{
			UpdateDC: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				return d
			},
			UpdateSVC: func(s *api.Service) *api.Service {
				return s
			},
			UpdateJob: func(j *batch.Job) *batch.Job {
				return nil
			},
			TestName:    "test setup data happy",
			ExpectError: false,
			Assert: func(d *dcapi.DeploymentConfig) error {
				container := d.Spec.Template.Spec.Containers[0]
				connURLFound := false
				for _, env := range container.Env {
					if env.Name == "FH_MONGODB_CONN_URL" {
						connURLFound = true
						if !strings.HasPrefix(env.Value, "mongodb://") {
							return fmt.Errorf("expected mongo url to have mongodb://")
						}
					}
				}
				if !connURLFound {
					return fmt.Errorf("failed to find env var FH_MONGODB_CONN_URL")
				}
				return nil
			},
			Calls: map[string]int{
				"FindDeploymentConfigsByLabel": 1,
				"FindServiceByLabel":           1,
				"FindJobByName":                1,
			},
		},
		{
			TestName:    "test setup data does not execute for data deployments",
			ExpectError: false,
			Assert: func(d *dcapi.DeploymentConfig) error {
				container := d.Spec.Template.Spec.Containers[0]
				for _, env := range container.Env {
					if env.Name == "FH_MONGODB_CONN_URL" {
						return fmt.Errorf("did not expect to find FH_MONGODB_CONN_URL")
					}
				}
				return nil
			},
			UpdateJob: func(j *batch.Job) *batch.Job {
				return nil
			},
			UpdateDC: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				d.Labels = map[string]string{"rhmap/name": "data-mongo"}
				d.Spec.Template.Spec.Containers[0].Env = []api.EnvVar{{
					Name:  "MONGODB_REPLICA_NAME",
					Value: "",
				}, {
					Name:  "MONGODB_ADMIN_PASSWORD",
					Value: "password",
				}}
				return d
			},
			UpdateSVC: func(s *api.Service) *api.Service {
				return s
			},
			Calls: map[string]int{
				"FindDeploymentConfigsByLabel": 0,
				"FindServiceByLabel":           0,
				"FindJobByName":                0,
			},
		},
		{
			TestName:    "test setup data does not execute when missing service",
			ExpectError: true,
			UpdateJob: func(j *batch.Job) *batch.Job {
				return nil
			},
			Assert: func(d *dcapi.DeploymentConfig) error {
				if d != nil {
					return fmt.Errorf("did not expect a DeploymentConfig")
				}
				return nil
			},
			UpdateDC: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				return nil
			},
			UpdateSVC: func(s *api.Service) *api.Service {
				return s
			},
			Calls: map[string]int{
				"FindDeploymentConfigsByLabel": 1,
				"FindServiceByLabel":           0,
				"FindJobByName":                1,
			},
		},
		{
			TestName:    "test setup data does not execute when job already exists",
			ExpectError: false,
			UpdateJob: func(j *batch.Job) *batch.Job {
				return j
			},
			Assert: func(d *dcapi.DeploymentConfig) error {
				if d == nil {
					return fmt.Errorf("expected to recieve a DeploymentConfig")
				}
				return nil
			},
			UpdateDC: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				return d
			},
			UpdateSVC: func(s *api.Service) *api.Service {
				return s
			},
			Calls: map[string]int{
				"FindDeploymentConfigsByLabel": 0,
				"FindServiceByLabel":           0,
				"FindJobByName":                1,
			},
		},
		{
			TestName:    "test setup data does not execute when missing kub svc def",
			ExpectError: true,
			Assert: func(d *dcapi.DeploymentConfig) error {
				if d != nil {
					return fmt.Errorf("did not expect a DeploymentConfig")
				}
				return nil
			},
			UpdateDC: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				return d
			},
			UpdateSVC: func(s *api.Service) *api.Service {
				return nil
			},
			UpdateJob: func(j *batch.Job) *batch.Job {
				return nil
			},
			Calls: map[string]int{
				"FindDeploymentConfigsByLabel": 1,
				"FindServiceByLabel":           1,
				"FindJobByName":                1,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			//setup the DeploymentConfig fresh for each test
			depConfig := []dcapi.DeploymentConfig{
				{ObjectMeta: api.ObjectMeta{Name: "data"}, Spec: dcapi.DeploymentConfigSpec{
					Template: &api.PodTemplateSpec{
						Spec: api.PodSpec{
							Containers: []api.Container{{
								Name: "",
								Env: []api.EnvVar{{
									Name:  "MONGODB_REPLICA_NAME",
									Value: "",
								}, {
									Name:  "MONGODB_ADMIN_PASSWORD",
									Value: "password",
								}},
							}},
						},
					},
				}},
			}
			service := api.Service{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"rhmap/name": "data",
					},
				},
			}
			job := tc.UpdateJob(&batch.Job{})
			client := mock.NewPassClient()
			dep := tc.UpdateDC(&depConfig[0])
			var depList = []dcapi.DeploymentConfig{}
			var svcList = []api.Service{}
			if dep != nil {
				depList = append(depList, *dep)
			}
			s := tc.UpdateSVC(&service)
			if nil != s {
				svcList = append(svcList, *s)
			}

			client.Returns["FindDeploymentConfigsByLabel"] = depList
			client.Returns["FindServiceByLabel"] = svcList
			client.Returns["FindJobByName"] = job
			// run our configure and test the result
			deployment, err := dataConfig.Configure(client, &depConfig[0], "test")
			if tc.ExpectError && err == nil {
				t.Fatalf(" expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatalf(" did not expect an error but got %s %+v", err.Error(), err)
			}
			for f, n := range tc.Calls {
				if n != client.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times", f, n)
				}
			}
			if err := tc.Assert(deployment); err != nil {
				t.Fatalf("assert error occurred %s ", err.Error())
			}
		})
	}

}
