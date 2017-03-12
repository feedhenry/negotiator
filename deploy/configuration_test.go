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

func (msp *mockStatusPublisher) Publish(key string, status deploy.ConfigurationStatus) {
	fmt.Println("calling mockstatus publisher", key, status.Status)
	msp.Statuses = append(msp.Statuses, status.Status)
	fmt.Println("status", msp.Statuses)
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
						}},
					}},
				},
			},
		}},
	}
	client.Returns["FindDeploymentConfigsByLabel"] = depConfig

	dispatched := deploy.Dispatched{Deployment: "test", NameSpace: "test"}
	if err := cacheConfig.Configure(client, &dispatched); err != nil {
		t.Fatalf("did not expect an error from cacheconfig but got %s ", err.Error())
	}
	t.Log(msp.Statuses)
	t.Log(depConfig)

}
