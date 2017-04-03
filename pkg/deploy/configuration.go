package deploy

import (
	"math/rand"
	"time"

	"github.com/pkg/errors"

	"strings"

	"fmt"

	"sync"

	"github.com/feedhenry/negotiator/pkg/log"
	dc "github.com/openshift/origin/pkg/deploy/api"
	"github.com/feedhenry/negotiator/pkg/config"
)

// LogStatusPublisher publishes the status to the log
type LogStatusPublisher struct {
	Logger log.Logger
}

// Publish is called to send something new to the log
func (lsp LogStatusPublisher) Publish(key string, status, description string) error {
	lsp.Logger.Info(key, status, description)
	return nil
}

func (lsp LogStatusPublisher) Clear(key string) error {
	return nil
}

// StatusKey returns a key for logging information against
func StatusKey(instanceID, operation string) string {
	return strings.Join([]string{instanceID, operation}, ":")
}

// ConfigurationFactory is responsible for finding the right implementation of the Configurer interface and returning it to all the configuration of the environment
type ConfigurationFactory struct {
	StatusPublisher StatusPublisher
	TemplateLoader  TemplateLoader
	Logger          log.Logger
}

// Publisher allows us to set the StatusPublisher for the Configurers
func (cf *ConfigurationFactory) Publisher(publisher StatusPublisher) {
	cf.StatusPublisher = publisher
}

// Factory is called to get a new Configurer based on the service type
func (cf *ConfigurationFactory) Factory(service string, config *Configuration, wait *sync.WaitGroup) Configurer {
	switch service {
	case templateCacheRedis:
		return &CacheRedisConfigure{
			StatusPublisher: cf.StatusPublisher,
			statusKey:       StatusKey(config.InstanceID, config.Action),
			wait:            wait,
		}
	case templateDataMongo:
		return &DataMongoConfigure{
			StatusPublisher: cf.StatusPublisher,
			TemplateLoader:  cf.TemplateLoader,
			logger:          cf.Logger,
			statusKey:       StatusKey(config.InstanceID, config.Action),
			wait:            wait,
		}
	case templateDataMysql:
		return &DataMysqlConfigure{
			StatusPublisher: cf.StatusPublisher,
			TemplateLoader:  cf.TemplateLoader,
			logger:          cf.Logger,
			statusKey:       StatusKey(config.InstanceID, config.Action),
			wait:            wait,
		}
	case templatePushUps:
		return &PushUpsConfigure{
			StatusPublisher: cf.StatusPublisher,
			TemplateLoader:  cf.TemplateLoader,
			logger:          cf.Logger,
		}
	}

	panic("unknown service type cannot configure")
}

// Status represent the current status of the configuration
type Status struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Log         []string  `json:"log"`
	Started     time.Time `json:"-"`
}

// StatusPublisher defines what a status publisher should implement
type StatusPublisher interface {
	Publish(key string, status, description string) error
	Clear(key string) error
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
	templateLoader       TemplateLoader
}

// NewEnvironmentServiceConfigController returns a new EnvironmentServiceConfigController
func NewEnvironmentServiceConfigController(configFactory ServiceConfigFactory, log log.Logger, publisher StatusPublisher, tl TemplateLoader) *EnvironmentServiceConfigController {
	if nil == publisher {
		publisher = LogStatusPublisher{Logger: log}
	}
	return &EnvironmentServiceConfigController{
		ConfigurationFactory: configFactory,
		StatusPublisher:      publisher,
		logger:               log,
		templateLoader:       tl,
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func genPass(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

const (
	configInProgress = "in progress"
	configError      = "failed"
	configComplete   = "succeeded"
)

// Configuration encapsulates information needed to configure a service
type Configuration struct {
	DeploymentName string
	NameSpace      string
	Action         string
	InstanceID     string
}

// Configure is called to configure the DeploymentConfig of a service that is currently being deployed
func (cac *EnvironmentServiceConfigController) Configure(client Client, config *Configuration) error {
	//cloudapp deployment config should be in place at this point, but check
	deploymentName := config.DeploymentName
	namespace := config.NameSpace
	statusKey := StatusKey(config.InstanceID, config.Action)
	waitGroup := &sync.WaitGroup{}
	// ensure we have the latest DeploymentConfig
	deployment, err := client.GetDeploymentConfigByName(namespace, deploymentName)
	if err != nil {
		return errors.Wrap(err, "unexpected error retrieving DeployConfig for deployment "+deploymentName)
	}
	if deployment == nil {
		return errors.New("could not find DeploymentConfig for " + deploymentName)
	}
	//find the deployed services
	services, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/type": "environmentService"})
	if err != nil {
		cac.StatusPublisher.Publish(statusKey, "error", "failed to retrieve environment Service dcs during configuration of  "+deployment.Name+" "+err.Error())
		return err
	}
	cac.StatusPublisher.Publish(statusKey, configInProgress, fmt.Sprintf("found %v services", len(services)))
	errs := []string{}
	//configure for any environment services already deployed
	// ensure not to call configure multiple times for instance when mongo replica set present
	configured := map[string]bool{}
	for _, s := range services {
		serviceName := s.Labels["rhmap/name"]
		if _, ok := configured[serviceName]; ok {
			continue
		}
		cac.StatusPublisher.Publish(statusKey, configInProgress, "configuring "+serviceName)
		configured[serviceName] = true
		c := cac.ConfigurationFactory.Factory(serviceName, config, waitGroup)
		go c.Configure(client, deployment, namespace)
	}
	go func() {
		waitGroup.Wait()
		if _, err := client.UpdateDeployConfigInNamespace(namespace, deployment); err != nil {
			cac.StatusPublisher.Publish(statusKey, configError, "failed to update DeployConfig after configuring it")
		}
		if len(errs) > 0 {
			cac.StatusPublisher.Publish(statusKey, configError, fmt.Sprintf(" some configuration jobs failed %v", errs))
		}
		cac.StatusPublisher.Publish(statusKey, configComplete, "service configuration complete")
	}()
	return nil
}

// PushUpsConfigure is an object for configuring push connection variables
type PushUpsConfigure struct {
	StatusPublisher StatusPublisher
	TemplateLoader  TemplateLoader
	logger          log.Logger
	statusKey       string
}

func (p *PushUpsConfigure) statusUpdate(description, status string) {
	if err := p.StatusPublisher.Publish(p.statusKey, status, description); err != nil {
		p.logger.Error("failed to publish status ", err.Error())
	}
}

// Configure the Push vars here
func (p *PushUpsConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {

	return deployment, nil
}

func waitForService(client Client, namespace, serviceName string) error {
	conf := config.Conf{}
	// poll deploy, waiting for success
	timeout := time.After(time.Second * time.Duration(conf.DependencyTimeout()))
	for {
		select {
		case <-timeout:
		//timed out, exit
			return errors.New("timed out waiting for dependency: " + serviceName + " to deploy")
		default:
			body, err := client.GetDeployLogs(namespace, serviceName)
			if err != nil {
				continue
			}
		// if success move on to configure job
			if strings.Contains(strings.ToLower(body), "success") {
				return nil
			}
		}
	}
}