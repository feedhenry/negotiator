package deploy_test

import (
	"sync"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
)

func TestMysqlConfiguration(t *testing.T) {
	tl := openshift.NewTemplateLoaderDecoder("../openshift/templates/")
	msp := &mockStatusPublisher{}
	logger := logrus.StandardLogger()
	factory := deploy.ConfigurationFactory{
		StatusPublisher: msp,
		Logger:          logger,
		TemplateLoader:  tl,
	}
	deployingConfig := func() dcapi.DeploymentConfig {
		return dcapi.DeploymentConfig{
			ObjectMeta: api.ObjectMeta{
				Name: "test",
				Labels: map[string]string{
					"rhmap/guid": "",
				},
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
	getMysqlsc := func() []api.Service {

		return []api.Service{
			{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"rhmap/name": "data-mysql",
						"rhmap/guid": "asdasdasdassdasdasdsadasdadasdasdasdasdsda",
					},
				},
			},
		}
	}

	mysqlJob := batch.Job{}

	cases := []struct {
		Name          string
		ExpectError   bool
		MysqlDC       func() []dcapi.DeploymentConfig
		MysqlSvc      func() []api.Service
		MysqlJob      func() *batch.Job
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
			MysqlDC: func() []dcapi.DeploymentConfig {
				return getMysqldc()
			},
			MysqlJob: func() *batch.Job {
				return nil
			},
			MysqlSvc: func() []api.Service {
				return getMysqlsc()
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
				foundMYSQL_HOST := false
				foundMYSQL_PORT := false
				foundMYSQL_SERVICE_PORT := false
				foundMYSQLUSER := false
				foundMYSQLPASSWORD := false
				foundMYSQLDATABASE := false
				for _, e := range env {
					if e.Name == "MYSQL_SERVICE_PORT" {
						foundMYSQL_SERVICE_PORT = true
					}
					if e.Name == "MYSQL_HOST" {
						foundMYSQL_HOST = true
					}
					if e.Name == "MYSQL_PORT" {
						foundMYSQL_PORT = true
					}
					if e.Name == "MYSQL_PASSWORD" {
						foundMYSQLPASSWORD = true
					}
					if e.Name == "MYSQL_DATABASE" {
						foundMYSQLDATABASE = true
					}
					if e.Name == "MYSQL_USER" {
						foundMYSQLUSER = true
					}
				}
				if !foundMYSQL_HOST && foundMYSQL_PORT && foundMYSQL_SERVICE_PORT && foundMYSQLDATABASE && foundMYSQLPASSWORD && foundMYSQLUSER {
					t.Fatalf("did not find MYSQL ENV VARS  but expected to")
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
			MysqlDC: func() []dcapi.DeploymentConfig {
				return getMysqldc()
			},
			MysqlJob: func() *batch.Job {
				return &mysqlJob
			},
			MysqlSvc: func() []api.Service {
				return getMysqlsc()
			},
			DcToConfigure: func() *dcapi.DeploymentConfig {
				dc := deployingConfig()
				dc.Spec.Template.Spec.Containers[0].Env = append(dc.Spec.Template.Spec.Containers[0].Env, api.EnvVar{
					Name:  "MYSQL_HOST",
					Value: "mysql",
				})
				dc.Spec.Template.Spec.Containers[0].Env = append(dc.Spec.Template.Spec.Containers[0].Env, api.EnvVar{
					Name:  "MYSQL_USER",
					Value: "test",
				})
				return &dc

			},
			AssertDC: func(t *testing.T, dc *dcapi.DeploymentConfig) {
				if nil == dc {
					t.Fatalf("did not expect the DeploymentConfig to be nil")
				}
				env := dc.Spec.Template.Spec.Containers[0].Env
				foundMYSQLUSER := false
				for _, e := range env {
					if e.Name == "MYSQL_USER" {
						foundMYSQLUSER = true
						if e.Value != "test" {
							t.Fatalf("expected the MYSQL_USER not to have changed but it did %s", e.Value)
						}
					}
				}
				if !foundMYSQLUSER {
					t.Fatalf("did not find MYSQL_USER but expected to")
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
			MysqlDC: func() []dcapi.DeploymentConfig {
				return getMysqldc()
			},
			MysqlJob: func() *batch.Job {
				return nil
			},
			MysqlSvc: func() []api.Service {
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
			MysqlDC: func() []dcapi.DeploymentConfig {
				return []dcapi.DeploymentConfig{}
			},
			MysqlJob: func() *batch.Job {
				return nil
			},
			MysqlSvc: func() []api.Service {
				return getMysqlsc()
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
			client.Returns["FindDeploymentConfigsByLabel"] = tc.MysqlDC()
			client.Returns["FindServiceByLabel"] = tc.MysqlSvc()
			client.Returns["FindJobByName"] = tc.MysqlJob()
			configure := factory.Factory("data-mysql", &deploy.Configuration{InstanceID: "instance", Action: "provision"}, &sync.WaitGroup{})
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
