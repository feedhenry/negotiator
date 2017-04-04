package deploy

import (
	"strings"
)

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
		dcs, err := client.FindDeploymentConfigsByLabel(nameSpace, map[string]string{"rhmap/name": dep})
		if len(dcs) > 0 {
			continue
		}
		dispatched, err := c.Template(client, dep, nameSpace, payload)
		if err != nil {
			return dependencies, err
		}
		dependencies = append(dependencies, dispatched)
	}

	return dependencies, nil
}
