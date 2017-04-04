package deploy_test

import (
	"testing"

	"sync"

	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
)

func TestMongoConfiguration(t *testing.T) {
	tl := openshift.NewTemplateLoaderDecoder("../openshift/templates/")
	msp := &mockStatusPublisher{}
	logger := logrus.StandardLogger()
	factory := deploy.ConfigurationFactory{
		StatusPublisher: msp,
		Logger:          logger,
		TemplateLoader:  tl,
	}
	mongodc := []dcapi.DeploymentConfig{
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
	mongosc := []api.Service{{
		ObjectMeta: api.ObjectMeta{
			Name: "data-mongo",
			Labels: map[string]string{
				"rhmap/name": "data-mongo",
			},
		}},
	}
	deployingConfig := func() dcapi.DeploymentConfig {
		return dcapi.DeploymentConfig{
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
							}},
						}},
					},
				},
			},
		}
	}

	mongojob := batch.Job{}

	cases := []struct {
		Name          string
		ExpectError   bool
		MongoDC       func() []dcapi.DeploymentConfig
		MongoSvc      func() []api.Service
		MongoJob      func() *batch.Job
		GetDeployLogs func() string
		DcToConfigure func() *dcapi.DeploymentConfig
		Namespace     string
		AssertDC      func(t *testing.T, dc *dcapi.DeploymentConfig)
		Calls         map[string]int
	}{
		{
			Name:        "test configure happy",
			Namespace:   "test",
			ExpectError: false,
			Calls: map[string]int{
				"FindJobByName":                1,
				"FindDeploymentConfigsByLabel": 1,
				"FindServiceByLabel":           1,
				"CreateJobToWatch":             1,
			},
			MongoDC: func() []dcapi.DeploymentConfig {
				return mongodc
			},
			MongoJob: func() *batch.Job {
				return nil
			},
			GetDeployLogs: func() string {
				return "Success"
			},
			MongoSvc: func() []api.Service {
				return mongosc
			},
			DcToConfigure: func() *dcapi.DeploymentConfig {
				dc := deployingConfig()
				return &dc
			},
			AssertDC: func(t *testing.T, dc *dcapi.DeploymentConfig) {
				if nil == dc {
					t.Fatalf("did not expect the DeploymentConfig to be nil")
				}
				env := dc.Spec.Template.Spec.Containers[0].Env
				foundMongoEnv := false
				for _, e := range env {
					if e.Name == "FH_MONGODB_CONN_URL" {
						foundMongoEnv = true
						mongoReg := regexp.MustCompile("mongodb:\\/\\/[\\w]+:[\\w]+@[\\w-]+:27017\\/[\\w]+")
						if !mongoReg.Match([]byte(e.Value)) {
							t.Fatalf("mongo url was not correct %s", e.Value)
						}
					}
				}
				if !foundMongoEnv {
					t.Fatalf("did not find FH_MONGODB_CONN_URL but expected to")
				}
			},
		},
		{
			Name:        "test does not continue when existing job found",
			Namespace:   "test",
			ExpectError: false,
			Calls: map[string]int{
				"FindJobByName":                1,
				"FindDeploymentConfigsByLabel": 0,
				"CreateJobToWatch":             0,
			},
			MongoDC: func() []dcapi.DeploymentConfig {
				return mongodc
			},
			MongoJob: func() *batch.Job {
				return &mongojob
			},
			GetDeployLogs: func() string {
				return "Success"
			},
			MongoSvc: func() []api.Service {
				return mongosc
			},
			DcToConfigure: func() *dcapi.DeploymentConfig {
				dc := deployingConfig()
				dc.Spec.Template.Spec.Containers[0].Env = append(dc.Spec.Template.Spec.Containers[0].Env, api.EnvVar{
					Name:  "FH_MONGODB_CONN_URL",
					Value: "mongodb://thisisatest",
				})
				return &dc

			},
			AssertDC: func(t *testing.T, dc *dcapi.DeploymentConfig) {
				if nil == dc {
					t.Fatalf("did not expect the DeploymentConfig to be nil")
				}
				env := dc.Spec.Template.Spec.Containers[0].Env
				foundMongoEnv := false
				for _, e := range env {
					if e.Name == "FH_MONGODB_CONN_URL" {
						foundMongoEnv = true
						if e.Value != "mongodb://thisisatest" {
							t.Fatalf("expected the mongo FH_MONGODB_CONN_URL not to have changed but it did %s", e.Value)
						}
					}
				}
				if !foundMongoEnv {
					t.Fatalf("did not find FH_MONGODB_CONN_URL but expected to")
				}
			},
		},
		{
			Name:        "test does not continue when no service found",
			Namespace:   "test",
			ExpectError: true,
			Calls: map[string]int{
				"FindJobByName":                1,
				"FindDeploymentConfigsByLabel": 1,
				"FindServiceByLabel":           1,
				"CreateJobToWatch":             0,
			},
			MongoDC: func() []dcapi.DeploymentConfig {
				return mongodc
			},
			MongoJob: func() *batch.Job {
				return nil
			},
			GetDeployLogs: func() string {
				return "Success"
			},
			MongoSvc: func() []api.Service {
				return []api.Service{}
			},
			DcToConfigure: func() *dcapi.DeploymentConfig {
				dc := deployingConfig()
				return &dc
			},
			AssertDC: func(t *testing.T, dc *dcapi.DeploymentConfig) {
				if nil != dc {
					t.Fatalf("expected the DeploymentConfig to be nil")
				}
			},
		},
		{
			Name:        "test does not continue when no service DeploymentConfig found",
			Namespace:   "test",
			ExpectError: true,
			Calls: map[string]int{
				"FindJobByName":                1,
				"FindDeploymentConfigsByLabel": 1,
				"FindServiceByLabel":           0,
				"CreateJobToWatch":             0,
			},
			MongoDC: func() []dcapi.DeploymentConfig {
				return []dcapi.DeploymentConfig{}
			},
			MongoJob: func() *batch.Job {
				return nil
			},
			GetDeployLogs: func() string {
				return "Success"
			},
			MongoSvc: func() []api.Service {
				return mongosc
			},
			DcToConfigure: func() *dcapi.DeploymentConfig {
				dc := deployingConfig()
				return &dc
			},
			AssertDC: func(t *testing.T, dc *dcapi.DeploymentConfig) {
				if nil != dc {
					t.Fatalf("expected the DeploymentConfig to be nil")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			client := mock.NewPassClient()
			client.Returns["FindDeploymentConfigsByLabel"] = tc.MongoDC()
			client.Returns["FindServiceByLabel"] = tc.MongoSvc()
			client.Returns["FindJobByName"] = tc.MongoJob()
			client.Returns["GetDeployLogs"] = tc.GetDeployLogs()
			configure := factory.Factory("data-mongo", &deploy.Configuration{InstanceID: "instance", Action: "provision"}, &sync.WaitGroup{})
			dc, err := configure.Configure(client, tc.DcToConfigure(), tc.Namespace)
			if tc.ExpectError && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatalf("did not expect an error but got one %s ", err.Error())
			}
			for f, n := range tc.Calls {
				if n != client.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times, it was called %d times", f, n, client.CalledTimes(f))
				}
			}
			tc.AssertDC(t, dc)

		})
	}
}
