package deploy

import (
	"time"

	"github.com/pkg/errors"

	"strings"

	"github.com/feedhenry/negotiator/pkg/log"
	dc "github.com/openshift/origin/pkg/deploy/api"
)

// LogStatusPublisher publishes the status to the log
type LogStatusPublisher struct {
	Logger log.Logger
}

// Publish is called to send something new to the log
func (lsp LogStatusPublisher) Publish(confifJobID string, status ConfigurationStatus) error {
	lsp.Logger.Info(confifJobID, status)
	return nil
}

// ConfigurationFactory is responsible for finding the right implementation of the Configurer interface and returning it to all the configuration of the environment
type ConfigurationFactory struct {
	StatusPublisher StatusPublisher
}

// Publisher allows us to set the StatusPublisher for the Configurers
func (cf *ConfigurationFactory) Publisher(publisher StatusPublisher) {
	cf.StatusPublisher = publisher
}

// Factory is called to get a new Configurer based on the service type
func (cf *ConfigurationFactory) Factory(service string) Configurer {
	if service == templateCache {
		return &CacheConfigure{
			StatusPublisher: cf.StatusPublisher,
		}
	}
	return nil
}

// ConfigurationStatus represent the current status of the configuration
type ConfigurationStatus struct {
	Status  string    `json:"status"`
	Log     []string  `json:"log"`
	Started time.Time `json:"-"`
}

// StatusPublisher defines what a status publisher should implement
type StatusPublisher interface {
	Publish(key string, status ConfigurationStatus) error
}

// Configurer defines what an environment Configurer should look like
type Configurer interface {
	Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error)
}

// EnvironmentServiceConfigController controlls the configuration of environments and services
type EnvironmentServiceConfigController struct {
	ConfigurationFactory ServiceConfigFactory
	StatusPublisher      StatusPublisher
	logger               log.Logger
}

// NewEnvironmentServiceConfigController returns a new EnvironmentServiceConfigController
func NewEnvironmentServiceConfigController(configFactory ServiceConfigFactory, log log.Logger, publisher StatusPublisher) *EnvironmentServiceConfigController {
	if nil == publisher {
		publisher = LogStatusPublisher{Logger: log}
	}
	return &EnvironmentServiceConfigController{
		StatusPublisher:      publisher,
		ConfigurationFactory: configFactory,
		logger:               log,
	}
}

// Configure is called to configure the DeploymentConfig of a service that is currently being deployed
func (cac *EnvironmentServiceConfigController) Configure(client Client, deploymentName, namespace string) error {
	//cloudapp deployment config should be in place at this point, but check
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{"starting configuration for service " + deploymentName}, Status: "inProgress"}
	key := namespace + "/" + deploymentName
	cac.StatusPublisher.Publish(key, configurationStatus)
	var statusUpdate = func(message, status string) {
		configurationStatus.Status = status
		configurationStatus.Log = append(configurationStatus.Log, message)
		cac.StatusPublisher.Publish(key, configurationStatus)
	}
	// ensure we have the latest DeploymentConfig
	deployment, err := client.GetDeploymentConfigByName(namespace, deploymentName)
	if err != nil {
		statusUpdate("unexpected error retrieving DeploymentConfig"+err.Error(), "error")
		return err
	}
	if deployment == nil {
		statusUpdate("could not find DeploymentConfig for "+deploymentName, "error")
		return errors.New("could not find DeploymentConfig for " + deploymentName)
	}
	//find the deployed services
	services, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/type": "environmentService"})
	if err != nil {
		statusUpdate("failed to retrieve environment Service dcs during configuration of  "+deployment.Name+" "+err.Error(), "error")
		return err
	}
	errs := []string{}
	//configure for any environment services already deployed
	for _, s := range services {
		serviceName := s.Labels["rhmap/name"]
		c := cac.ConfigurationFactory.Factory(serviceName)
		if nil == c{
			cac.logger.Info("no configurer for service " + serviceName)
			continue
		}
		if _, err := c.Configure(client, deployment, namespace); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if _, err := client.UpdateDeployConfigInNamespace(namespace, deployment); err != nil {
		return errors.Wrap(err, "failed to update deployment after configuring it ")
	}
	if len(errs) > 0 {
		return errors.New("some services failed to configure: " + strings.Join(errs, " : "))
	}
	return nil
}

// CacheConfigure is a Configurer for the cache service
type CacheConfigure struct {
	StatusPublisher StatusPublisher
}

// Configure configures the current DeploymentConfig with the need configuration to use cache
func (c *CacheConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{"starting configuration"}, Status: "inProgress"}
	c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	var statusUpdate = func(message, status string) {
		configurationStatus.Status = status
		configurationStatus.Log = append(configurationStatus.Log, message)
		c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	}
	// likely needs to be broken out as it will be needed for all services
	statusUpdate("updating containers env for deployment "+deployment.GetName(), "inProgress")
	for ci := range deployment.Spec.Template.Spec.Containers {
		env := deployment.Spec.Template.Spec.Containers[ci].Env
		for ei, e := range env {
			if e.Name == "FH_REDIS_HOST" && e.Value != "cache" {
				deployment.Spec.Template.Spec.Containers[ci].Env[ei].Value = "cache" //hard coded for time being
				break
			}
		}
	}
	return deployment, nil
}
