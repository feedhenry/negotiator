package deploy

import (
	"errors"
	"strings"
	"sync"
	"time"
	"github.com/Sirupsen/logrus"
)

//noinspection GoVarDeclaration
func deployDependencyServices(c Controller, client Client, template *Template, nameSpace string, payload *Payload) ([]*Dispatched, error) {
	if _, ok := template.Annotations["dependencies"]; !ok {
		// no dependencies to process
		return nil, nil
	}

	if strings.TrimSpace(template.Annotations["dependencies"]) == "" {
		return nil, nil
	}

	dependencies := []*Dispatched{}

	for _, dep := range strings.Split(template.Annotations["dependencies"], " ") {

		dep = strings.ToLower(strings.TrimSpace(dep))
		// Trim out common bad data
		if dep == "" || dep == "," {
			continue
		}

		// check if dependency is already deployed
		logrus.Info("Checking for service: rhmap/name: " + dep + " in namespace: " + nameSpace)
		dcs, err := client.FindDeploymentConfigsByLabel(nameSpace, map[string]string{"rhmap/name": dep})
		if len(dcs) > 0 {
			logrus.Info("service found, skipping this dependency")
			continue
		}
		logrus.Info("Service not found, deploying this dependency")
		dispatched, err := c.Template(client, dep, nameSpace, payload)
		if err != nil {
			return dependencies, err
		}
		dependencies = append(dependencies, dispatched)
	}

	return dependencies, nil
}

func waitForDependencies(client Client, namespace string, dependencies []*Dispatched, payload *Payload) error {
	var dependencyGroup sync.WaitGroup
	depErrors := []string{}
	for _, dependency := range dependencies {
		dependencyGroup.Add(1)
		go func(dependency *Dispatched) {
			defer dependencyGroup.Done()
			// poll deploy for 5 minutes, waiting for success
			timeout := 300
			start := time.Now().UTC().Second()
			for {
				body, err := client.GetDeployLogs(namespace, dependency.DeploymentName)
				if err != nil {
					continue
				}
				// if success exit
				if strings.Contains(strings.ToLower(body), "success") {
					return
				}
				//timed out, exit
				if time.Now().UTC().Second()-start > timeout {
					depErrors = append(depErrors, "Failed to deploy dependency: "+dependency.DeploymentName)
				}
			}

		}(dependency)
	}

	dependencyGroup.Wait()

	// dependencies were not successful, return an error
	if len(depErrors) > 0 {
		return errors.New(strings.Join(depErrors, "\n"))
	}

	return nil
}
