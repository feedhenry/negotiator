package deploy_test

import (
	"fmt"
	"testing"

	"strings"

	"sync"

	"github.com/Sirupsen/logrus"
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

func (msp *mockStatusPublisher) Publish(key string, status, description string) error {
	msp.Statuses = append(msp.Statuses, status)
	return nil
}

func (msp *mockStatusPublisher) Clear(key string) error {
	return nil
}

func TestConfigure(t *testing.T) {
	t.Skip("STILL NEED TO WRITE THIS TEST ")
}

func TestDataConfigurationJob(t *testing.T) {
	tl := openshift.NewTemplateLoaderDecoder("../openshift/templates/")
	msp := &mockStatusPublisher{}
	logger := logrus.StandardLogger()
	factory := deploy.ConfigurationFactory{
		StatusPublisher: msp,
		Logger:          logger,
		TemplateLoader:  tl,
	}
	getDataMongoConfig := func() deploy.Configurer {
		return factory.Factory("data-mongo", &deploy.Configuration{InstanceID: "test", Action: "provision"}, &sync.WaitGroup{})
	}
	getDataMySQLConfig := func() deploy.Configurer {
		return factory.Factory("data-mysql", &deploy.Configuration{InstanceID: "test", Action: "provision"}, &sync.WaitGroup{})
	}

	getMongodc := func() []dcapi.DeploymentConfig {
		return []dcapi.DeploymentConfig{
			{ObjectMeta: api.ObjectMeta{Name: "data-mongo"}, Spec: dcapi.DeploymentConfigSpec{
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
	}
	getMongosc := func() api.Service {
		return api.Service{
			ObjectMeta: api.ObjectMeta{
				Labels: map[string]string{
					"rhmap/name": "data-mongo",
				},
			},
		}
	}
	//setup the MySQL DeploymentConfig fresh for each test
	getMysqldc := func() []dcapi.DeploymentConfig {
		return []dcapi.DeploymentConfig{
			{
				ObjectMeta: api.ObjectMeta{
					Name: "data-mysql",
					Labels: map[string]string{
						"rhmap/guid": "asdasdasdassdasdasdsadasdadasdasdasdasdsda",
					},
				},
				Spec: dcapi.DeploymentConfigSpec{
					Template: &api.PodTemplateSpec{
						Spec: api.PodSpec{
							Containers: []api.Container{{
								Name: "",
								Env: []api.EnvVar{
									{
										Name:  "MYSQL_ROOT_PASSWORD",
										Value: "dfgdfgdf",
									},
								},
							}},
						},
					},
				}},
		}
	}
	getMysqlsc := func() api.Service {
		return api.Service{
			ObjectMeta: api.ObjectMeta{
				Labels: map[string]string{
					"rhmap/name": "data-mysql",
					"rhmap/guid": "asdasdasdassdasdasdsadasdadasdasdasdasdsda",
				},
			},
		}
	}

	cases := []struct {
		TestName    string
		ExpectError bool
		Assert      func(d *dcapi.DeploymentConfig) error
		UpdateDC    func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig
		UpdateSVC   func(d *api.Service) *api.Service
		UpdateJob   func(j *batch.Job) *batch.Job
		GetDC       func() []dcapi.DeploymentConfig
		GetSC       func() api.Service
		GetConfig   func() deploy.Configurer
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
			GetConfig:   getDataMongoConfig,
			GetDC:       getMongodc,
			GetSC:       getMongosc,
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
			UpdateDC: func(d *dcapi.DeploymentConfig) *dcapi.DeploymentConfig {
				return d
			},
			UpdateSVC: func(s *api.Service) *api.Service {
				return s
			},
			UpdateJob: func(j *batch.Job) *batch.Job {
				return nil
			},
			TestName:    "test setup data mysql happy",
			GetConfig:   getDataMySQLConfig,
			GetDC:       getMysqldc,
			GetSC:       getMysqlsc,
			ExpectError: false,
			Assert: func(d *dcapi.DeploymentConfig) error {
				container := d.Spec.Template.Spec.Containers[0]
				userFound := false
				passFound := false
				databaseFound := false
				for _, env := range container.Env {
					if env.Name == "MYSQL_USER" {
						userFound = true
					} else if env.Name == "MYSQL_PASSWORD" {
						passFound = true
					} else if env.Name == "MYSQL_DATABASE" {
						databaseFound = true
					}
				}
				if !userFound {
					return fmt.Errorf("failed to find env var MYSQL_USER")
				}
				if !passFound {
					return fmt.Errorf("failed to find env var MYSQL_PASSWORD")
				}
				if !databaseFound {
					return fmt.Errorf("failed to find env var MYSQL_DATABASE")
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
			GetConfig:   getDataMongoConfig,
			GetDC:       getMongodc,
			GetSC:       getMongosc,
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
			GetConfig:   getDataMongoConfig,
			GetDC:       getMongodc,
			GetSC:       getMongosc,
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
			GetConfig:   getDataMongoConfig,
			GetDC:       getMongodc,
			GetSC:       getMongosc,
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
			GetConfig:   getDataMongoConfig,
			GetDC:       getMongodc,
			GetSC:       getMongosc,
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
			depConfig := tc.GetDC()
			service := tc.GetSC()
			dataConfig := tc.GetConfig()
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
					t.Errorf("Expected %s to be called %d times, it was called %d times", f, n, client.CalledTimes(f))
				}
			}
			if err := tc.Assert(deployment); err != nil {
				t.Fatalf("assert error occurred %s ", err.Error())
			}
		})
	}

}
