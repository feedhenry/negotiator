package deploy_test

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	route "github.com/openshift/origin/pkg/route/api"
)

func TestUPSConfigure(t *testing.T) {
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
	UPSdc := func() []dcapi.DeploymentConfig {
		return []dcapi.DeploymentConfig{
			{
				ObjectMeta: api.ObjectMeta{
					Name: "push-ups",
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
										Name:  "ADMIN_USER",
										Value: "admin",
									},
									{
										Name:  "ADMIN_PASSWORD",
										Value: "123",
									},
								},
							}},
						},
					},
				}},
		}
	}
	UPSsvc := func() []api.Service {

		return []api.Service{
			{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"rhmap/name": "push-ups",
						"rhmap/guid": "asdasdasdassdasdasdsadasdadasdasdasdasdsda",
					},
				},
			},
		}
	}

	UPSRoute := func() route.Route {
		return route.Route{
			ObjectMeta: api.ObjectMeta{Name: "push-ups"},
			Spec: route.RouteSpec{
				Host: "http://ups.pushy.com",
			},
		}
	}

	cases := []struct {
		Name          string
		ExpectError   bool
		UPSDC         func() []dcapi.DeploymentConfig
		UPSSvc        func() []api.Service
		UPSRoute      func() route.Route
		DcToConfigure func() *dcapi.DeploymentConfig
		Namespace     string
		AssertDC      func(t *testing.T, dc *dcapi.DeploymentConfig)
		Calls         map[string]int
		PushLister    func(host, user, pass string) ([]*deploy.PushApplication, error)
	}{
		{
			Name:        "test ups configure happy",
			ExpectError: false,
			UPSDC:       UPSdc,
			UPSSvc:      UPSsvc,
			UPSRoute:    UPSRoute,
			PushLister: func(host, user, pass string) ([]*deploy.PushApplication, error) {

				return []*deploy.PushApplication{
					&deploy.PushApplication{
						Name:              "testapp",
						MasterSecret:      "secret",
						PushApplicationID: "id",
					},
				}, nil
			},
			DcToConfigure: func() *dcapi.DeploymentConfig {
				dc := deployingConfig()
				return &dc
			},
			Namespace: "test",
			AssertDC: func(t *testing.T, dc *dcapi.DeploymentConfig) {
				if nil == dc {
					t.Fatalf("did not expect the DeploymentConfig to nil")
				}
				vmounts := dc.Spec.Template.Spec.Containers[0].VolumeMounts
				volumes := dc.Spec.Template.Spec.Volumes
				if len(vmounts) != 1 {
					t.Fatalf("expected 1 volume mount but got %v", len(vmounts))
				}
				if vmounts[0].Name != volumes[0].Name {
					t.Fatalf("expected to get %s but got %s ", volumes[0].Name, vmounts[0].Name)
				}
				if len(volumes) != 1 {
					t.Fatalf("expected 1 volume but got %v ", len(volumes))
				}
				upsServiceEnv := false
				upsConfigEnv := false
				for _, e := range dc.Spec.Template.Spec.Containers[0].Env {
					if e.Name == "UPS_SERVICE_HOST" {
						upsServiceEnv = true
					}
					if e.Name == "UPS_CONFIG_PATH" {
						upsConfigEnv = true
					}
				}
				if !upsConfigEnv || !upsServiceEnv {
					t.Fatalf("missing env var expected to find UPS_CONFIG_PATH and UPS_SERVICE_HOST")
				}
			},
			Calls: map[string]int{
				"FindDeploymentConfigsByLabel": 1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			client := mock.NewPassClient()
			client.Returns["FindDeploymentConfigsByLabel"] = tc.UPSDC()

			configure := factory.Factory("push-ups", &deploy.Configuration{InstanceID: "instanceID", Action: "provision"}, &sync.WaitGroup{})
			pConfigure := configure.(*deploy.PushUpsConfigure)
			pConfigure.PushLister = tc.PushLister
			deployed, err := pConfigure.Configure(client, tc.DcToConfigure(), tc.Namespace)
			if tc.ExpectError && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatalf("did not expect an error but got %s ", err.Error())
			}
			for f, n := range tc.Calls {
				if n != client.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times, it was called %d times", f, n, client.CalledTimes(f))
				}
			}
			tc.AssertDC(t, deployed)
		})
	}

}
