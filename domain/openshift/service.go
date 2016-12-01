package openshift

import (
	bc "github.com/openshift/origin/pkg/build/api"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
)

//domain logic for creating a cloud app in openshift

// PaaSClient is the interface this controller expects for interacting with an openshift paas
type PaaSClient interface {
	ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error)
	CreateServiceInNamespace(ns string, svc *api.Service) (*api.Service, error)
	CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error)
	CreateImageStream(ns string, i *ioapi.ImageStream) (*ioapi.ImageStream, error)
	CreateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error)
}

// Service encapsulates the domain logic for deploying our app to OpenShift
type Service struct {
	client PaaSClient
}

// NewService return an instance of the OpenShift Service
func NewService(client PaaSClient) Service {
	return Service{
		client: client,
	}
}

// CreateRoute defines a route for a given cloud app
func (s Service) CreateRoute(namespace, serviceToBindTo, appName, optionalHost string, labels map[string]string) error {
	if nil == labels {
		labels = map[string]string{}
	}
	route := &roapi.Route{
		ObjectMeta: api.ObjectMeta{
			Name:   appName,
			Labels: labels,
		},
		Spec: roapi.RouteSpec{
			Host: optionalHost,
			To: roapi.RouteTargetReference{
				Kind: "Service",
				Name: serviceToBindTo,
			},
			TLS: &roapi.TLSConfig{
				Termination:                   roapi.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: roapi.InsecureEdgeTerminationPolicyType("Allow"),
			},
		},
	}
	if _, err := s.client.CreateRouteInNamespace(namespace, route); err != nil {
		return err
	}

	return nil
}

// CreateService sets up a service via the Kubernetes API
func (s Service) CreateService(namespace, serviceName, selector, description string, port int32, labels map[string]string) (*api.Service, error) {
	serv := &api.Service{
		ObjectMeta: api.ObjectMeta{
			Name:   serviceName,
			Labels: labels,
			Annotations: map[string]string{
				"rhmap/description": description,
				"rhmap/title":       selector,
				"description":       "round robin loadbalancer for your app",
			},
		},
		Spec: api.ServiceSpec{
			Ports: []api.ServicePort{
				{
					Name:       "web",
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
				},
			},
			Selector: map[string]string{
				"name": selector,
			},
		},
	}
	retServ, err := s.client.CreateServiceInNamespace(namespace, serv)
	if err != nil {
		return nil, err
	}

	return retServ, nil
}

/**
{
        "kind": "ImageStream",
        "apiVersion": "v1",
        "metadata": {
          "name": nodejsObjectsTitle,
          "labels" : labels,
          "annotations" : {
            "rhmap/description" : description,
            "rhmap/title" : title,
            "description": "Keeps track of changes in the application image"
          }
        }
      }

*/
// CreateImageStream setup an image stream for the cloud app
func (s Service) CreateImageStream(namespace, name string, labels map[string]string) error {
	is := &ioapi.ImageStream{
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: labels,
			Annotations: map[string]string{
				"rhmap/description": "image stream for tracking changes to your app",
				"rhmap/title":       name,
				"description":       "",
			},
		},
	}
	if _, err := s.client.CreateImageStream(namespace, is); err != nil {
		return err
	}
	return nil
}

/**
{
    "apiVersion": "v1",
    "kind": "Secret",
    "type": "Opaque",
    "metadata": {
      "name": `${nodejsObjectsTitle}-scmsecret`,
      "labels" : labels,
      "annotations" : {
        "rhmap/description" : description,
        "rhmap/title" : title,
        "description": "SSH keypair used to clone the application from a private repo"
      }
    }
	"data":{
		username:username,
		password:password
	}
  }
 **/
// CreateSecret creates a secret in the OSCP
func (s Service) CreateSecret(namespace, name string) error {

	return nil
}

// CreateBuildConfig creates a buildconfig in the OSCP
func (s Service) CreateBuildConfig(namespace, name, selector, description, gitUrl, gitBranch string, labels map[string]string) error {
	bc := &bc.BuildConfig{
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: labels,
			Annotations: map[string]string{
				"rhmap/description": description,
				"rhmap/title":       selector,
				"description":       "round robin loadbalancer for your app",
			},
		},
		Spec: bc.BuildConfigSpec{
			RunPolicy: bc.BuildRunPolicySerialLatestOnly,
			CommonSpec: bc.CommonSpec{
				Source: bc.BuildSource{
					Git: &bc.GitBuildSource{
						URI: gitUrl,
						Ref: gitBranch,
					},
					SourceSecret: &api.LocalObjectReference{
						Name: name + "-scmsecret",
					},
				},
				Strategy: bc.BuildStrategy{
					SourceStrategy: &bc.SourceBuildStrategy{
						From: api.ObjectReference{
							Kind:      "ImageStreamTag",
							Namespace: namespace,
							Name:      selector + ":latest",
						},
						Env: []api.EnvVar{
							{
								Name:  "NODE_ENV",
								Value: "production",
							},
						},
					},
				},
				Output: bc.BuildOutput{
					To: &api.ObjectReference{
						Kind: "ImageStreamTag",
						Name: name + ":latest",
					},
				},
			},
			Triggers: []bc.BuildTriggerPolicy{
				{
					Type: "ImageChange",
				},
			},
		},
	}

	if _, err := s.client.CreateBuildConfigInNamespace(namespace, bc); err != nil {
		return err
	}

	return nil
}

// DeploymentConfig

/**
{
        "kind": "DeploymentConfig",
        "apiVersion": "v1",
        "metadata": {
          "name": nodejsObjectsTitle,
          "labels" : labels,
          "annotations" : {
            "rhmap/description" : description,
            "rhmap/title" : title,
            "description": "Defines how to deploy the application server"
          }
        },
        "spec": {
          "strategy": {
            "type": "Rolling"
          },
          "triggers": [
            {
              "type": "ImageChange",
              "imageChangeParams": {
                "automatic": true,
                "containerNames": [
                  nodejsObjectsTitle
                ],
                "from": {
                  "kind": "ImageStreamTag",
                  "name": `${nodejsObjectsTitle}:latest`
                }
              }
            },
            {
              "type": "ConfigChange"
            }
          ],
          "replicas": 1,
          "selector": {
            "name": nodejsObjectsTitle
          },
          "template": {
            "metadata": {
              "name": nodejsObjectsTitle,
              "labels": {
                "name": nodejsObjectsTitle,
                "rhmap/env": labels["rhmap/env"],
                "rhmap/guid": labels["rhmap/guid"],
                "rhmap/domain": labels["rhmap/domain"],
                "rhmap/project": labels["rhmap/project"]
              }
            },
            "spec": {
              "containers": [
                {
                  "name": nodejsObjectsTitle,
                  "image": nodejsObjectsTitle,
                  "ports": [
                    {
                      "containerPort": 8001
                    }
                  ],
                  "env": formatEnvVarArray(params, redisServiceName),
                  "resources": {
                    "limits": {
                      "cpu": "500m",
                      "memory": "250Mi"
                    },
                    "requests": {
                      "cpu": "100m",
                      "memory": "90Mi"
                    }
                  }
                }
              ]
            }
          }
        }
      }
**/
// CreateDeploymentConfig creates a deploymentconfig in the OSCP
func (s Service) CreateDeploymentConfig(namespace, name string) error {
	return nil
}
