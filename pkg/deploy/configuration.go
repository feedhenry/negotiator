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
	Logger          log.Logger
}

// Publisher allows us to set the StatusPublisher for the Configurers
func (cf *ConfigurationFactory) Publisher(publisher StatusPublisher) {
	cf.StatusPublisher = publisher
}

// Factory is called to get a new Configurer based on the service type
func (cf *ConfigurationFactory) Factory(service string) Configurer {
	switch service {
	case templateCacheRedis:
		return &CacheRedisConfigure{
			StatusPublisher: cf.StatusPublisher,
		}
	case templateDataMongo:
		return &DataMongoConfigure{
			StatusPublisher: cf.StatusPublisher,
			TemplateLoader:  cf.TemplateLoader,
			logger:          cf.Logger,
		}
	case templateDataMysql:
		return &DataMysqlConfigure{
			StatusPublisher: cf.StatusPublisher,
			TemplateLoader:  cf.TemplateLoader,
			logger:          cf.Logger,
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
	// ensure not to call configure multiple times for instance when mongo replica set present
	configured := map[string]bool{}
	for _, s := range services {
		serviceName := s.Labels["rhmap/name"]
		if _, ok := configured[serviceName]; ok {
			continue
		}
		configured[serviceName] = true
		c := cac.ConfigurationFactory.Factory(serviceName)
		_, err := c.Configure(client, deployment, namespace)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if _, err := client.UpdateDeployConfigInNamespace(namespace, deployment); err != nil {
		return errors.Wrap(err, "failed to update deployment after configuring it ")
	}
	//TODO given we have a status updater do we really need to return errors from the configuration handlers
	if len(errs) > 0 {
		return errors.New("some services failed to configure: " + strings.Join(errs, " : "))
	}
	return nil
}

// CacheRedisConfigure is a Configurer for the cache service
type CacheRedisConfigure struct {
	StatusPublisher StatusPublisher
}

// Configure configures the current DeploymentConfig with the need configuration to use cache
func (c *CacheRedisConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
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
			if e.Name == "FH_REDIS_HOST" && e.Value != "data-cache" {
				deployment.Spec.Template.Spec.Containers[ci].Env[ei].Value = "data-cache" //hard coded for time being
				break
			}
		}
	}
	return deployment, nil
}

// DataMongoConfigure is a object for configuring mongo connection strings
type DataMongoConfigure struct {
	StatusPublisher StatusPublisher
	TemplateLoader  TemplateLoader
	status          *ConfigurationStatus
	logger          log.Logger
}

// DataMysqlConfigure is a object for configuring mysql connection variables
type DataMysqlConfigure struct {
	StatusPublisher StatusPublisher
	TemplateLoader  TemplateLoader
	status          *ConfigurationStatus
	logger          log.Logger
}

func (d *DataMongoConfigure) statusUpdate(key, message, status string) {
	if d.status == nil {
		d.status = &ConfigurationStatus{Started: time.Now(), Log: []string{}}
	}
	d.status.Log = append(d.status.Log, message)
	d.status.Status = status
	if err := d.StatusPublisher.Publish(key, *d.status); err != nil {
		d.logger.Info("failed to publish status", err.Error())
	}
}

// Configure takes an apps DeployConfig and sets of a job to create a new user and database in the mongodb data service. It also sets the expected env var FH_MONGODB_CONN_URL on the apps DeploymentConfig so it can be used to connect to the data service
func (d *DataMongoConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	esName := "data-mongo"
	d.statusUpdate(deployment.Name, "starting configuration of data service for "+deployment.Name, configInProgress)
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == esName {
			d.statusUpdate(deployment.Name, "no need to configure data-mongo DeploymentConfig ", configComplete)
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
	// look for the Job if it already exists no need to run it again
	existingJob, err := client.FindJobByName(namespace, deployment.Name+"-dataconfig-job")
	if err != nil {
		d.statusUpdate(deployment.Name, "error finding existing Job "+err.Error(), "error")
		return deployment, nil
	}
	if existingJob != nil {
		d.statusUpdate(deployment.Name, "configuration job "+deployment.Name+"-dataconfig-job already exists. No need to run again ", "complete")
		return deployment, nil
	}
	dataDc, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/name": esName})
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to find data DeploymentConfig. Cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataDc) == 0 {
		err := errors.New("no data DeploymentConfig exists. Cannot continue")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}
	dataService, err := client.FindServiceByLabel(namespace, map[string]string{"rhmap/name": esName})
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to find data service cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataService) == 0 {
		err := errors.New("no service for data found. Cannot continue")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}
	// if we get this far then the job does not exists so we will run another one which will update the FH_MONGODB_CONN_URL and create or update any database and user password definitions
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
		d.statusUpdate(deployment.Name, err.Error(), configError)
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

	tpl, err := d.TemplateLoader.Load("data-mongo-job")
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to load job template "+err.Error(), configError)
		return nil, errors.Wrap(err, "failed to load template data-mongo-job ")
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "data-mongo-job", jobOpts); err != nil {
		err = errors.Wrap(err, "failed to execute template: ")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}
	j := &batch.Job{}
	if err := runtime.DecodeInto(k8api.Codecs.UniversalDecoder(), buf.Bytes(), j); err != nil {
		err = errors.Wrap(err, "failed to Decode job")
		d.statusUpdate(deployment.Name, err.Error(), "error")
		return nil, err
	}
	w, err := client.CreateJobToWatch(j, namespace)
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to CreateJobToWatch "+err.Error(), configError)
		return nil, err
	}
	//set off job and watch it till complete
	go func() {
		result := w.ResultChan()
		for ws := range result {
			switch ws.Type {
			case watch.Added, watch.Modified:
				j := ws.Object.(*batch.Job)
				// succeeded will always be 1 if a deadline is reached
				if j.Status.Succeeded >= 1 {
					w.Stop()
					for _, condition := range j.Status.Conditions {
						if condition.Reason == "DeadlineExceeded" && condition.Type == "Failed" {
							d.statusUpdate(deployment.Name, "configuration job  timed out and failed to configure database  "+condition.Message, configError)
							//TODO Maybe we should delete the job a this point to allow it to be retried.
						} else if condition.Type == "Complete" {
							d.statusUpdate(deployment.Name, "configuration job succeeded ", configComplete)
						}
					}
				}
				d.statusUpdate(deployment.Name, fmt.Sprintf("job status succeeded %d failed %d", j.Status.Succeeded, j.Status.Failed), configInProgress)
			case watch.Error:
				d.statusUpdate(deployment.Name, " data-mongo configuration job error ", configError)
				//TODO maybe pull back the log from the pod here? also remove the job in this condition so it can be retried
				w.Stop()
			}

		}
	}()

	return deployment, nil
}

func (d *DataMysqlConfigure) statusUpdate(key, message, status string) {
	if d.status == nil {
		d.status = &ConfigurationStatus{Started: time.Now(), Log: []string{}}
	}
	d.status.Log = append(d.status.Log, message)
	d.status.Status = status
	if err := d.StatusPublisher.Publish(key, *d.status); err != nil {
		d.logger.Info("failed to publish status", err.Error())
	}
}

// Configure the mysql connection vars here
func (d *DataMysqlConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	d.statusUpdate(deployment.Name, "starting configuration of data service for "+deployment.Name, configInProgress)
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == templateDataMysql {
			d.statusUpdate(deployment.Name, "no need to configure data-mysql DeploymentConfig ", configComplete)
			return deployment, nil
		}
	}
	dataDc, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/name": templateDataMysql})
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to find data DeploymentConfig. Cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataDc) == 0 {
		err := errors.New("no data DeploymentConfig exists. Cannot continue")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}
	// look for the Job if it already exists no need to run it again
	existingJob, err := client.FindJobByName(namespace, deployment.Name+"-dataconfig-job")
	if err != nil {
		d.statusUpdate(deployment.Name, "error finding existing Job "+err.Error(), "error")
		return deployment, nil
	}
	if existingJob != nil {
		d.statusUpdate(deployment.Name, "configuration job "+deployment.Name+"-dataconfig-job already exists. No need to run again ", "complete")
		return deployment, nil
	}
	dataService, err := client.FindServiceByLabel(namespace, map[string]string{"rhmap/name": templateDataMysql})
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to find data service cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataService) == 0 {
		err := errors.New("no service for data found. Cannot continue")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}
	jobOpts := map[string]interface{}{}

	containerEnv := dataDc[0].Spec.Template.Spec.Containers[0].Env

	found := false
	for _, e := range containerEnv {
		if e.Name == "MYSQL_ROOT_PASSWORD" {
			jobOpts["admin-password"] = e.Value
			found = true
			break
		}
	}
	if !found {
		err := errors.New("expected to find an env var: MYSQL_ROOT_PASSWORD but it was not present")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}

	jobName := "data-mysql-job"
	jobOpts["name"] = deployment.Name
	jobOpts["dbhost"] = dataService[0].GetName()

	jobOpts["admin-username"] = "root"
	jobOpts["admin-database"] = "mysql"

	if v, ok := deployment.Labels["rhmap/guid"]; ok {
		jobOpts["user-database"] = v
	} else {
		return nil, errors.New("Could not find rhmap/guid for deployment: " + deployment.Name)
	}
	jobOpts["user-password"] = genPass(16)
	jobOpts["user-username"] = jobOpts["user-database"].(string)[:16]

	for ci := range deployment.Spec.Template.Spec.Containers {
		env := deployment.Spec.Template.Spec.Containers[ci].Env
		envFromOpts := map[string]string{
			"MYSQL_USER":     "user-username",
			"MYSQL_PASSWORD": "user-password",
			"MYSQL_DATABASE": "user-database",
		}
		for envName, optsName := range envFromOpts {
			found := false
			for ei, e := range env {
				if e.Name == envName {
					deployment.Spec.Template.Spec.Containers[ci].Env[ei].Value = jobOpts[optsName].(string)
					found = true
					break
				}
			}
			if !found {
				deployment.Spec.Template.Spec.Containers[ci].Env = append(deployment.Spec.Template.Spec.Containers[ci].Env, k8api.EnvVar{
					Name:  envName,
					Value: jobOpts[optsName].(string),
				})
			}
		}

	}
	tpl, err := d.TemplateLoader.Load(jobName)
	if err != nil {
		d.statusUpdate(deployment.Name, "failed to load job template "+err.Error(), configError)
		return nil, errors.Wrap(err, "failed to load template "+jobName)
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, jobName, jobOpts); err != nil {
		err = errors.Wrap(err, "failed to execute template: ")
		d.statusUpdate(deployment.Name, err.Error(), configError)
		return nil, err
	}
	j := &batch.Job{}
	if err := runtime.DecodeInto(k8api.Codecs.UniversalDecoder(), buf.Bytes(), j); err != nil {
		err = errors.Wrap(err, "failed to Decode job")
		d.statusUpdate(deployment.Name, err.Error(), "error")
		return nil, err
	}
	//set off job and watch it till complete
	go func() {
		w, err := client.CreateJobToWatch(j, namespace)
		if err != nil {
			d.statusUpdate(deployment.Name, "failed to CreateJobToWatch "+err.Error(), configError)
			return
		}
		result := w.ResultChan()
		for ws := range result {
			switch ws.Type {
			case watch.Added, watch.Modified:
				j := ws.Object.(*batch.Job)
				if j.Status.Succeeded >= 1 {
					d.statusUpdate(deployment.Name, "configuration job succeeded ", configComplete)
					w.Stop()
				}
				d.statusUpdate(deployment.Name, fmt.Sprintf("job status succeeded %d failed %d", j.Status.Succeeded, j.Status.Failed), configInProgress)
				for _, condition := range j.Status.Conditions {
					if condition.Reason == "DeadlineExceeded" {
						d.statusUpdate(deployment.Name, "configuration job failed to configure database in time "+condition.Message, configError)
						w.Stop()
					}
				}
			case watch.Error:
				d.statusUpdate(deployment.Name, " data configuration job error ", configError)
				w.Stop()
			}

		}
	}()

	return deployment, nil
}
