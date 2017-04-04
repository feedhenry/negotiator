package deploy

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/feedhenry/negotiator/pkg/log"

	dc "github.com/openshift/origin/pkg/deploy/api"
	"github.com/pkg/errors"
	k8api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

// DataMongoConfigure is a object for configuring mongo connection strings
type DataMongoConfigure struct {
	StatusPublisher StatusPublisher
	statusKey       string
	TemplateLoader  TemplateLoader
	status          *Status
	logger          log.Logger
	wait            *sync.WaitGroup
}

func (d *DataMongoConfigure) statusUpdate(description, status string) {
	if err := d.StatusPublisher.Publish(d.statusKey, status, description); err != nil {
		d.logger.Error("failed to publish status ", err.Error())
	}
}

// Configure takes an apps DeployConfig and sets of a job to create a new user and database in the mongodb data service. It also sets the expected env var FH_MONGODB_CONN_URL on the apps DeploymentConfig so it can be used to connect to the data service
func (d *DataMongoConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	esName := "data-mongo"
	d.wait.Add(1)
	defer d.wait.Done()
	d.statusUpdate("starting configuration of "+esName+" service for "+deployment.Name, configInProgress)
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == esName {
			d.statusUpdate("no need to configure "+esName+" DeploymentConfig ", configInProgress)
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
	// look for the Job if it already exists no need to run it again. TODO we could consider checking the job status and retrying if it has failed
	existingJob, err := client.FindJobByName(namespace, deployment.Name+"-dataconfig-job")
	if err != nil {
		d.statusUpdate("error finding existing Job "+err.Error(), "error")
		return deployment, nil
	}
	if existingJob != nil {
		d.statusUpdate("configuration job "+deployment.Name+"-dataconfig-job has already executed. No need to run again ", configInProgress)
		return deployment, nil
	}
	dataDc, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/name": esName})
	if err != nil {
		d.statusUpdate("failed to find data DeploymentConfig. Cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataDc) == 0 {
		err := errors.New("no data DeploymentConfig exists. Cannot continue")
		d.statusUpdate(err.Error(), configError)
		return nil, err
	}
	dataService, err := client.FindServiceByLabel(namespace, map[string]string{"rhmap/name": esName})
	if err != nil {
		d.statusUpdate("failed to find data service cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataService) == 0 {
		err := errors.New("no service for data found. Cannot continue")
		d.statusUpdate(err.Error(), configError)
		return nil, err
	}

	//block here until the service is ready to accept a connection from this job
	err = waitForService(client, namespace, templateDataMongo)
	if err != nil {
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
		d.statusUpdate(err.Error(), configError)
		return nil, err
	}
	jobOpts["dbhost"] = dataService[0].GetName()
	jobOpts["admin-user"] = "admin"
	jobOpts["database"] = deployment.Name
	jobOpts["name"] = deployment.Name
	if v, ok := deployment.Labels["rhmap/guid"]; ok {
		if v != "" {
			jobOpts["database"] = v
		}
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
		d.statusUpdate("failed to load job template "+err.Error(), configError)
		return nil, errors.Wrap(err, "failed to load template data-mongo-job ")
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "data-mongo-job", jobOpts); err != nil {
		err = errors.Wrap(err, "failed to execute template: ")
		d.statusUpdate(err.Error(), configError)
		return nil, err
	}
	j := &batch.Job{}
	if err := runtime.DecodeInto(k8api.Codecs.UniversalDecoder(), buf.Bytes(), j); err != nil {
		err = errors.Wrap(err, "failed to Decode job")
		d.statusUpdate(err.Error(), "error")
		return nil, err
	}
	w, err := client.CreateJobToWatch(j, namespace)
	if err != nil {
		d.statusUpdate("failed to CreateJobToWatch "+err.Error(), configError)
		return nil, err
	}
	//set off job and watch it till complete
	go func() {
		d.wait.Add(1)
		defer d.wait.Done()
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
							d.statusUpdate("configuration job  timed out and failed to configure database  "+condition.Message, configError)
							//TODO Maybe we should delete the job a this point to allow it to be retried.
						} else if condition.Type == "Complete" {
							d.statusUpdate("configuration job succeeded ", configInProgress)
						}
					}
				}
				d.statusUpdate(fmt.Sprintf("job status succeeded %d failed %d", j.Status.Succeeded, j.Status.Failed), configInProgress)
			case watch.Error:
				d.statusUpdate(" data-mongo configuration job error ", configError)
				//TODO maybe pull back the log from the pod here? also remove the job in this condition so it can be retried
				w.Stop()
			}

		}
	}()

	return deployment, nil
}
