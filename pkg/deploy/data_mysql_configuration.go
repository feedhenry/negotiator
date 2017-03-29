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

// DataMysqlConfigure is a object for configuring mysql connection variables
type DataMysqlConfigure struct {
	StatusPublisher StatusPublisher
	statusKey       string
	TemplateLoader  TemplateLoader
	logger          log.Logger
	wait            *sync.WaitGroup
}

func (d *DataMysqlConfigure) statusUpdate(description, status string) {
	if err := d.StatusPublisher.Publish(d.statusKey, status, description); err != nil {
		d.logger.Error("failed to publish status", err.Error())
	}
}

// Configure the mysql connection vars here
func (d *DataMysqlConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	d.wait.Add(1)
	defer d.wait.Done()
	d.statusUpdate("starting configuration of data service for "+deployment.Name, configInProgress)
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == templateDataMysql {
			d.statusUpdate("no need to configure data-mysql DeploymentConfig ", configInProgress)
			return deployment, nil
		}
	}
	// look for the Job if it already exists no need to run it again
	existingJob, err := client.FindJobByName(namespace, deployment.Name+"-dataconfig-job")
	if err != nil {
		d.statusUpdate("error finding existing Job "+err.Error(), "error")
		return deployment, nil
	}
	if existingJob != nil {
		d.statusUpdate("configuration job "+deployment.Name+"-dataconfig-job already exists. No need to run again ", "complete")
		return deployment, nil
	}
	dataDc, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/name": templateDataMysql})
	if err != nil {
		d.statusUpdate("failed to find data DeploymentConfig. Cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataDc) == 0 {
		err := errors.New("no data DeploymentConfig exists. Cannot continue")
		d.statusUpdate(err.Error(), configError)
		return nil, err
	}
	dataService, err := client.FindServiceByLabel(namespace, map[string]string{"rhmap/name": templateDataMysql})
	if err != nil {
		d.statusUpdate("failed to find data service cannot continue "+err.Error(), configError)
		return nil, err
	}
	if len(dataService) == 0 {
		err := errors.New("no service for data found. Cannot continue")
		d.statusUpdate(err.Error(), configError)
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
		d.statusUpdate(err.Error(), configError)
		return nil, err
	}

	jobName := "data-mysql-job"
	jobOpts["name"] = deployment.Name
	jobOpts["dbhost"] = dataService[0].GetName()

	jobOpts["admin-username"] = "root"
	jobOpts["admin-database"] = "mysql"

	if v, ok := deployment.Labels["rhmap/guid"]; ok {
		if v == "" {
			// this is unique to the environment
			v = deployment.Name
		}
		jobOpts["user-database"] = v
	} else {
		return nil, errors.New("Could not find rhmap/guid for deployment: " + deployment.Name)
	}
	jobOpts["user-password"] = genPass(16)
	databaseUser := jobOpts["user-database"].(string)
	jobOpts["user-username"] = databaseUser
	if len(databaseUser) > 16 {
		jobOpts["user-username"] = databaseUser[:16]
	}
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
		deployment.Spec.Template.Spec.Containers[ci].Env = append(deployment.Spec.Template.Spec.Containers[ci].Env, k8api.EnvVar{
			Name:  "MYSQL_HOST",
			Value: dataService[0].GetName(),
		})
		deployment.Spec.Template.Spec.Containers[ci].Env = append(deployment.Spec.Template.Spec.Containers[ci].Env, k8api.EnvVar{
			Name:  "MYSQL_PORT",
			Value: "3306",
		})
		deployment.Spec.Template.Spec.Containers[ci].Env = append(deployment.Spec.Template.Spec.Containers[ci].Env, k8api.EnvVar{
			Name:  "MYSQL_SERVICE_PORT",
			Value: "3306",
		})
	}
	tpl, err := d.TemplateLoader.Load(jobName)
	if err != nil {
		d.statusUpdate("failed to load job template "+err.Error(), configError)
		return nil, errors.Wrap(err, "failed to load template "+jobName)
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, jobName, jobOpts); err != nil {
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
	//set off job and watch it till complete
	w, err := client.CreateJobToWatch(j, namespace)
	if err != nil {
		d.statusUpdate("failed to CreateJobToWatch "+err.Error(), configError)
		return nil, errors.Wrap(err, "failed to CreateJobToWatch while configuring mysql")
	}
	go func() {
		d.wait.Add(1)
		defer d.wait.Done()

		result := w.ResultChan()
		for ws := range result {
			switch ws.Type {
			case watch.Added, watch.Modified:
				j := ws.Object.(*batch.Job)
				if j.Status.Succeeded >= 1 {
					d.statusUpdate("configuration job succeeded ", configInProgress)
					w.Stop()
				}
				d.statusUpdate(fmt.Sprintf("job status succeeded %d failed %d", j.Status.Succeeded, j.Status.Failed), configInProgress)
				for _, condition := range j.Status.Conditions {
					if condition.Reason == "DeadlineExceeded" {
						d.statusUpdate("configuration job failed to configure database in time "+condition.Message, configError)
						w.Stop()
					}
				}
			case watch.Error:
				d.statusUpdate(" data configuration job error ", configError)
				w.Stop()
			}

		}
	}()
	return deployment, nil
}
