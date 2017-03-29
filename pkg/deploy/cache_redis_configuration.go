package deploy

import (
	"sync"

	dc "github.com/openshift/origin/pkg/deploy/api"
)

// CacheRedisConfigure is a Configurer for the cache service
type CacheRedisConfigure struct {
	StatusPublisher StatusPublisher
	statusKey       string
	wait            *sync.WaitGroup
}

// Configure configures the current DeploymentConfig with the need configuration to use cache
func (c *CacheRedisConfigure) Configure(client Client, deployment *dc.DeploymentConfig, namespace string) (*dc.DeploymentConfig, error) {
	c.wait.Add(1)
	defer c.wait.Done()
	c.StatusPublisher.Publish(c.statusKey, configInProgress, "starting configuration of cache ")
	if v, ok := deployment.Labels["rhmap/name"]; ok {
		if v == "cache" {
			c.StatusPublisher.Publish(c.statusKey, configInProgress, "no need to configure own DeploymentConfig ")
			return deployment, nil
		}
	}
	// likely needs to be broken out as it will be needed for all services
	c.StatusPublisher.Publish(c.statusKey, configInProgress, "updating containers env for deployment "+deployment.GetName())
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
