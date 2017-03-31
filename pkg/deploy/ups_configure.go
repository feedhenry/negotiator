package deploy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"time"

	"github.com/feedhenry/negotiator/pkg/log"
	dc "github.com/openshift/origin/pkg/deploy/api"
	"github.com/pkg/errors"
	k8api "k8s.io/kubernetes/pkg/api"
)

// PushUpsConfigure is an object for configuring push connection variables
type PushUpsConfigure struct {
	StatusPublisher StatusPublisher
	TemplateLoader  TemplateLoader
	logger          log.Logger
	statusKey       string
	PushLister      func(host, user, password string) ([]*PushApplication, error)
}

func (p *PushUpsConfigure) statusUpdate(description, status string) {
	if err := p.StatusPublisher.Publish(p.statusKey, status, description); err != nil {
		p.logger.Error("failed to publish status ", err.Error())
	}
}

// DefaultPushLister list PushApplications from ups
var DefaultPushLister = func(host, user, password string) ([]*PushApplication, error) {
	hc := http.Client{}
	hc.Timeout = time.Second * 10
	form := url.Values{}
	form.Add("grant_type", "password")
	form.Add("username", user)
	form.Add("password", password)
	form.Add("client_id", "unified-push-server-js")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/realms/aerogear/protocol/openid-connect/token", host), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to listPushApplications ")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request to listPushApplications ")
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body listPushApplications ")
	}
	payload := map[string]interface{}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal listPushApplications ")
	}
	token := payload["access_token"].(string)

	req, err = http.NewRequest("GET", fmt.Sprintf("%s/ag-push/rest/applications", host), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request listPushApplications ")
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-type", "application/json")
	resp, err = hc.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request to listPushApplications ")
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response listPushApplications ")
	}
	apps := []*PushApplication{}
	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal listPushApplications ")
	}

	return apps, nil
}

type PushApplication struct {
	PushApplicationID string `json:"pushApplicationID"`
	MasterSecret      string `json:"masterSecret"`
	Name              string `json:"name"`
}

// Configure the Push vars here
func (p *PushUpsConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	pushConfigPath := "/opt/rh/push-config"
	// find the push route
	route, err := client.FindRouteByName(namespace, "push-ups")
	if err != nil {
		return deployment, errors.Wrap(err, "failed to find route in Configure ")
	}
	p.StatusPublisher.Publish(p.statusKey, configInProgress, "found ups route")
	upsDcs, err := client.FindDeploymentConfigsByLabel(namespace, map[string]string{"rhmap/name": "push-ups"})
	if err != nil {
		return deployment, errors.Wrap(err, "failed to find ups-push DeploymentConfig")
	}
	if len(upsDcs) == 0 {
		return deployment, errors.New("ups-push DeploymentConfig not found")
	}
	upsService, err := client.FindServiceByLabel(namespace, map[string]string{"rhmap/name": "push-ups"})
	if err != nil || len(upsService) == 0 {
		if err != nil {
			return deployment, errors.Wrap(err, "failed to FindServiceByLabel during PushUpsConfigure "+err.Error())
		}
		return deployment, errors.New("failed to FindServiceByLabel during PushUpsConfigure")
	}

	p.StatusPublisher.Publish(p.statusKey, configInProgress, "found ups deployment")
	configMap, err := client.FindConfigMapByName(namespace, "ups-client-config")
	if err != nil {
		return deployment, errors.Wrap(err, "failed to FindConfigMapByName in PushUpsConfigure")
	}
	p.StatusPublisher.Publish(p.statusKey, configInProgress, "found ups client configMap")
	var user, pass string
	env := upsDcs[0].Spec.Template.Spec.Containers[0].Env
	for _, e := range env {
		if e.Name == "ADMIN_USER" {
			user = e.Value
		}
		if e.Name == "ADMIN_PASSWORD" {
			pass = e.Value
		}
	}
	p.StatusPublisher.Publish(p.statusKey, configInProgress, "found ups user pass "+user+":"+pass)
	//TODO if tls is edge switch to https
	host := "%s%s"
	if route.Spec.TLS == nil {
		host = fmt.Sprintf(host, "http://", route.Spec.Host)
	} else {
		host = fmt.Sprintf(host, "https://", route.Spec.Host)
	}
	apps, err := p.PushLister(host, user, pass)
	if err != nil {
		return deployment, errors.Wrap(err, "failed PushLister in PushUpsConfigure")
	}
	p.StatusPublisher.Publish(p.statusKey, configInProgress, fmt.Sprintf("found ups apps %v", len(apps)))
	configmapData := map[string]string{}
	appsMap := map[string]*PushApplication{}
	for _, a := range apps {
		appsMap[a.Name] = a
	}
	data, err := json.Marshal(appsMap)
	configmapData["config.json"] = string(data)
	configMap.Data = configmapData
	if _, err := client.UpdateConfigMap(namespace, configMap); err != nil {
		return deployment, errors.Wrap(err, "failed to UpdateConfigMap in PushUpsConfigure")
	}
	p.StatusPublisher.Publish(p.statusKey, configInProgress, "updated ups configmap")
	//check if volume already exists
	volumeFound := false
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		if v.Name == "push-config-volume" {
			volumeFound = true
			break
		}
	}
	// update env var for ups service
	for i := range deployment.Spec.Template.Spec.Containers {
		upsEnvFound := false
		pushPathFound := false
		for ie, e := range deployment.Spec.Template.Spec.Containers[i].Env {
			if e.Name == "UPS_SERVICE_HOST" { // dunno if we need these as kubernetes injects them
				deployment.Spec.Template.Spec.Containers[i].Env[ie].Value = upsService[0].Name
				upsEnvFound = true
			}
			if e.Name == "UPS_CONFIG_PATH" {
				pushPathFound = true
			}
		}
		if !upsEnvFound {
			deployment.Spec.Template.Spec.Containers[i].Env = append(deployment.Spec.Template.Spec.Containers[i].Env, k8api.EnvVar{
				Name:  "UPS_SERVICE_HOST",
				Value: "push-ups",
			})
		}
		if !pushPathFound {
			deployment.Spec.Template.Spec.Containers[i].Env = append(deployment.Spec.Template.Spec.Containers[i].Env, k8api.EnvVar{
				Name:  "UPS_CONFIG_PATH",
				Value: pushConfigPath + "/config.json",
			})
		}
	}
	// if we found the volume we are done
	if volumeFound {
		return deployment, nil
	}

	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, k8api.Volume{
		Name: "push-config-volume",
		VolumeSource: k8api.VolumeSource{
			ConfigMap: &k8api.ConfigMapVolumeSource{
				LocalObjectReference: k8api.LocalObjectReference{Name: "ups-client-config"},
			},
		},
	})
	for i := range deployment.Spec.Template.Spec.Containers {
		vmounts := deployment.Spec.Template.Spec.Containers[i].VolumeMounts
		vmFound := false
		for _, vm := range vmounts {
			if vm.Name == "push-config-volume" {
				vmFound = true
				break
			}
		}
		if !vmFound {
			deployment.Spec.Template.Spec.Containers[i].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[i].VolumeMounts, k8api.VolumeMount{
				Name:      "push-config-volume",
				MountPath: pushConfigPath,
				ReadOnly:  true,
			})
		}
	}

	return deployment, nil
}
