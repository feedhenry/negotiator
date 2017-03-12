package deploy

import (
	"fmt"

	"time"

	"github.com/pkg/errors"

	"github.com/feedhenry/negotiator/pkg/log"
	dc "github.com/openshift/origin/pkg/deploy/api"
)

type LogStatusPublisher struct {
	Logger log.Logger
}

func (lsp LogStatusPublisher) Publish(confifJobID string, status ConfigurationStatus) error {
	lsp.Logger.Info(confifJobID, status)
	return nil
}

type ConfigurationFactory struct {
	StatusPublisher StatusPublisher
}

var unconfigurableError = errors.New("cannot configure that service type")

type ConfigurationObserver interface {
	Observe(chan ConfigurationStatus)
}

func (cf ConfigurationFactory) Factory(service string) Configurer {
	if service == templateCloudApp {
		return &CloudAppConfigure{
			ConfigurationFactory: cf,
			StatusPublisher:      cf.StatusPublisher,
		}
	}
	if service == templateCache {
		return &CacheConfigure{
			StatusPublisher: cf.StatusPublisher,
		}
	}
	return nil
}

type ConfigurationStatus struct {
	Status  string    `json:"status"`
	Log     []string  `json:"log"`
	Started time.Time `json:"-"`
}

type StatusPublisher interface {
	Publish(key string, status ConfigurationStatus) error
}

type Configurer interface {
	Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error)
}

type CloudAppConfigure struct {
	//Bizzare I know probably needs a rethink
	ConfigurationFactory ConfigurationFactory
	StatusPublisher      StatusPublisher
}

func (cac *CloudAppConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	//cloudapp deployment config should be in place at this point, but check
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{"starting configuration"}, Status: "inProgress"}
	cac.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	var statusUpdate = func(message, status string) {
		configurationStatus.Status = status
		configurationStatus.Log = append(configurationStatus.Log, message)
		cac.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	}
	services, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/type": "environmentService"})
	if err != nil {
		statusUpdate("failed to retrieve environment Service dcs during configuration of cloud app "+deployment.Name+" "+err.Error(), "error")
		return nil, err
	}
	//configure the app for any services already deployed
	for _, s := range services {
		serviceName := s.Labels["rhmap/name"]
		c := cac.ConfigurationFactory.Factory(serviceName)
		c.Configure(client, deployment, namespace)
	}
	return deployment, nil
}

type CacheConfigure struct {
	StatusPublisher StatusPublisher
}

func (c *CacheConfigure) modifySingleDcForCloudApp(deployment *dc.DeploymentConfig, namespace string) {
	env := deployment.Spec.Template.Spec.Containers[0].Env
	for i, e := range env {
		if e.Name == "FH_REDIS_HOST" && e.Value == "" {
			deployment.Spec.Template.Spec.Containers[0].Env[i].Value = "cache" //hard coded for time being
			break
		}
	}
}

func (c *CacheConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	var searCritera = map[string]string{}
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{"starting configuration"}, Status: "inProgress"}
	c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)

	var statusUpdate = func(message, status string) {
		configurationStatus.Status = status
		configurationStatus.Log = append(configurationStatus.Log, message)
		c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	}

	if availableEnvironmentServices.isEnvironmentService(deployment.Name) {
		searCritera["rhmap/type"] = "cloudapp"
		cloudApps, err := client.FindDeploymentConfigsByLabel(namespace, searCritera)
		if err != nil {
			statusUpdate("unable to configure environment missing deployment context", "error")
			return nil, err
		}
		configurationStatus.Log = append(configurationStatus.Log, fmt.Sprintf("found %d apps to configure ", len(cloudApps)))
		c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
		for i := range cloudApps {
			d := cloudApps[i]
			c.modifySingleDcForCloudApp(&d, namespace)
			if _, err := client.UpdateDeployConfigInNamespace(namespace, &d); err != nil {
				statusUpdate("unable to configure client app DeploymentConfig "+err.Error(), "error")
			}
		}
		return deployment, nil
	}
	c.modifySingleDcForCloudApp(deployment, namespace)
	return nil, nil
}
