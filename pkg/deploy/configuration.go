package deploy

import (
	"math/rand"
	"time"

	"github.com/pkg/errors"

	"strings"

	"fmt"

	"bytes"

	"github.com/feedhenry/negotiator/pkg/log"
	dc "github.com/openshift/origin/pkg/deploy/api"
	k8api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
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
	TemplateLoader  TemplateLoader
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
	if service == templateData {
		return &DataConfigure{
			StatusPublisher: cf.StatusPublisher,
			TemplateLoader:  cf.TemplateLoader,
		}
	}
	panic("unknown service type cannot configure")
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
	templateLoader       TemplateLoader
}

// NewEnvironmentServiceConfigController returns a new EnvironmentServiceConfigController
func NewEnvironmentServiceConfigController(configFactory ServiceConfigFactory, log log.Logger, publisher StatusPublisher, tl TemplateLoader) *EnvironmentServiceConfigController {
	if nil == publisher {
		publisher = LogStatusPublisher{Logger: log}
	}
	return &EnvironmentServiceConfigController{
		StatusPublisher:      publisher,
		ConfigurationFactory: configFactory,
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
	configInProgress = "inProgress"
	configError      = "error"
	configComplete   = "complete"
)

// Configure is called to configure the DeploymentConfig of a service that is currently being deployed
func (cac *EnvironmentServiceConfigController) Configure(client Client, deploymentName, namespace string) error {
	//cloudapp deployment config should be in place at this point, but check
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{"starting configuration for service " + deploymentName}, Status: configInProgress}
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
		statusUpdate("unexpected error retrieving DeploymentConfig"+err.Error(), configError)
		return err
	}
	if deployment == nil {
		statusUpdate("could not find DeploymentConfig for "+deploymentName, configError)
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
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{"starting configuration"}, Status: configInProgress}
	c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	var statusUpdate = func(message, status string) {
		configurationStatus.Status = status
		configurationStatus.Log = append(configurationStatus.Log, message)
		c.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	}
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == "cache" {
			statusUpdate("no need to configure own DeploymentConfig ", configComplete)
			return deployment, nil
		}
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

type DataConfigure struct {
	StatusPublisher StatusPublisher
	TemplateLoader  TemplateLoader
}

// Configure takes an apps DeployConfig and sets of a job to create a new user and database in the mongodb data service. It also sets the expected env var FH_MONGODB_CONN_URL on the apps DeploymentConfig so it can be used to connect to the data service
func (d *DataConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	var configurationStatus = ConfigurationStatus{Started: time.Now(), Log: []string{}}
	var statusUpdate = func(message, status string) {
		configurationStatus.Status = status
		configurationStatus.Log = append(configurationStatus.Log, message)
		d.StatusPublisher.Publish(deployment.GetResourceVersion(), configurationStatus)
	}
	statusUpdate("starting configuration of data service for "+deployment.Name, configInProgress)
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == "data" {
			statusUpdate("no need to configure own DeploymentConfig ", configComplete)
			return deployment, nil
		}
	}
	var constructMongoURL = func(host, user, pass, db, replicaSet interface{}) string {
		url := fmt.Sprintf("mongodb://%s:%s@%s:27017/%s", user.(string), pass.(string), host.(string), db.(string))
		if "" != replicaSet {
			url += "?replicaSet=" + replicaSet.(string)
		}
		return url
	}
	dataDc, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/name": "data"})
	if err != nil {
		statusUpdate("failed to find data DeploymentConfig. Cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataDc) == 0 {
		err := errors.New("no data DeploymentConfig exists. Cannot continue")
		statusUpdate(err.Error(), configError)
		return nil, err
	}
	dataService, err := client.FindServiceByLabel(namespace, map[string]string{"rhmap/name": "data"})
	if err != nil {
		statusUpdate("failed to find data service cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataService) == 0 {
		err := errors.New("no service for data found. Cannot continue")
		statusUpdate(err.Error(), configError)
		return nil, err
	}
	jobOpts := map[string]interface{}{}
	//we know we have a data deployment config and it will have 1 container
	containerEnv := dataDc[0].Spec.Template.Spec.Containers[0].Env
	foundAdminPassword := false
	for _, e := range containerEnv {
		if e.Name == "MONGODB_ADMIN_PASSWORD" {
			foundAdminPassword = true
			jobOpts["admin-pass"] = e.Value
		}
		if e.Name == "MONGODB_REPLICA_NAME" {
			jobOpts["replicaSet"] = e.Value
		}
	}
	if !foundAdminPassword {
		err := errors.New("expected to find an admin password but there was non present")
		statusUpdate(err.Error(), configError)
		return nil, err
	}
	jobOpts["dbhost"] = dataService[0].GetName()
	jobOpts["admin-user"] = "admin"
	jobOpts["database"] = deployment.Name
	jobOpts["name"] = deployment.Name
	if v, ok := deployment.Labels["rhmap/guid"]; ok {
		jobOpts["database"] = v
	}
	jobOpts["database-pass"] = genPass(16)
	jobOpts["database-user"] = jobOpts["database"]
	mongoURL := constructMongoURL(jobOpts["dbhost"], jobOpts["database-user"], jobOpts["database-pass"], jobOpts["database"], jobOpts["replicaSet"])
	for ci := range deployment.Spec.Template.Spec.Containers {
		env := deployment.Spec.Template.Spec.Containers[ci].Env
		found := false
		for ei, e := range env {
			if e.Name == "FH_MONGODB_CONN_URL" {
				deployment.Spec.Template.Spec.Containers[ci].Env[ei].Value = mongoURL
				found = true
				break
			}
		}
		if !found {
			deployment.Spec.Template.Spec.Containers[ci].Env = append(deployment.Spec.Template.Spec.Containers[ci].Env, k8api.EnvVar{
				Name:  "FH_MONGODB_CONN_URL",
				Value: mongoURL,
			})
		}
	}
	tpl, err := d.TemplateLoader.Load("data-job")
	if err != nil {
		statusUpdate("failed to load job template "+err.Error(), configError)
		return nil, errors.Wrap(err, "failed to load template data-job ")
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "data-job", jobOpts); err != nil {
		err = errors.Wrap(err, "failed to execute template: ")
		statusUpdate(err.Error(), configError)
		return nil, err
	}
	j := &batch.Job{}
	if err := runtime.DecodeInto(k8api.Codecs.UniversalDecoder(), buf.Bytes(), j); err != nil {
		err = errors.Wrap(err, "failed to Decode job")
		statusUpdate(err.Error(), "error")
		return nil, err
	}
	//set off job and watch it till complete
	go func() {
		w, err := client.CreateJobToWatch(j, namespace)
		if err != nil {
			statusUpdate("failed to CreateJobToWatch "+err.Error(), configError)
			return
		}
		result := w.ResultChan()
		for ws := range result {
			switch ws.Type {
			case watch.Added, watch.Modified:
				j := ws.Object.(*batch.Job)
				if j.Status.Succeeded >= 1 {
					statusUpdate("configuration job succeeded ", configComplete)
					w.Stop()
				}
				statusUpdate(fmt.Sprintf("job status succeeded %d failed %d", j.Status.Succeeded, j.Status.Failed), configInProgress)
				for _, condition := range j.Status.Conditions {
					if condition.Reason == "DeadlineExceeded" {
						statusUpdate("configuration job failed to configure database in time "+condition.Message, configError)
						w.Stop()
					}
				}
			case watch.Error:
				statusUpdate(" data configuration job error ", configError)
				w.Stop()
			}

		}
	}()

	return deployment, nil
}
