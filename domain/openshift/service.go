package openshift

import (
	"github.com/feedhenry/negotiator/controller"
	bc "github.com/openshift/origin/pkg/build/api"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
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
	CreateDeployConfigInNamespace(namespace string, d *dcapi.DeploymentConfig) (*dcapi.DeploymentConfig, error)
	CreateSecretInNamespace(namespace string, s *api.Secret) (*api.Secret, error)
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

func formatEnvVars(dcEnvVars []*controller.EnvVar) []api.EnvVar {
	envVars := []api.EnvVar{}
	for _, envVar := range dcEnvVars {
		envVars = append(envVars, api.EnvVar{Value: envVar.Value, Name: envVar.Key})
	}
	return envVars
}

// CreateRoute defines a route for a given cloud app
func (s Service) CreateRoute(dc controller.DeployCmd, optionalHost string) error {
	route := &roapi.Route{
		ObjectMeta: api.ObjectMeta{
			Name:   dc.AppName(),
			Labels: dc.Labels(),
		},
		Spec: roapi.RouteSpec{
			Host: optionalHost,
			To: roapi.RouteTargetReference{
				Kind: "Service",
				Name: dc.AppName(),
			},
			Port: &roapi.RoutePort{
				TargetPort: intstr.FromString("web"),
			},
			TLS: &roapi.TLSConfig{
				Termination:                   roapi.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: roapi.InsecureEdgeTerminationPolicyType("Allow"),
			},
		},
	}
	if _, err := s.client.CreateRouteInNamespace(dc.EnvironmentName(), route); err != nil {
		return err
	}

	return nil
}

// CreateService sets up a service via the Kubernetes API
func (s Service) CreateService(dc controller.DeployCmd, description string, port int32) error {
	serv := &api.Service{
		ObjectMeta: api.ObjectMeta{
			Name:   dc.AppName(),
			Labels: dc.Labels(),
			Annotations: map[string]string{
				"rhmap/description": description,
				"rhmap/title":       dc.AppName(),
				"description":       "Service for " + dc.AppName(),
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
				"name": dc.AppName(),
			},
		},
	}
	if _, err := s.client.CreateServiceInNamespace(dc.EnvironmentName(), serv); err != nil {
		return err
	}
	return nil
}

// CreateImageStream setup an image stream for the cloud app
func (s Service) CreateImageStream(dc controller.DeployCmd) error {
	is := &ioapi.ImageStream{
		ObjectMeta: api.ObjectMeta{
			Name:   dc.AppName(),
			Labels: dc.Labels(),
			Annotations: map[string]string{
				"rhmap/description": "image stream for tracking changes to your app",
				"rhmap/title":       dc.AppName(),
				"description":       "",
			},
		},
	}
	if _, err := s.client.CreateImageStream(dc.EnvironmentName(), is); err != nil {
		return err
	}
	return nil
}

// CreateSecret creates a secret in the OSCP
func (s Service) CreateSecret(dc controller.DeployCmd, data map[string][]byte) error {
	secret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name:   dc.AppName(),
			Labels: dc.Labels(),
			Annotations: map[string]string{
				"rhmap/description": "SSH kepair used to clone the application from a private repo",
				"rhmap/title":       dc.AppName(),
				"description":       "SSH keypair used to clone the application from a private repo",
			},
		},
		Data: data,
		Type: api.SecretTypeOpaque,
	}

	if _, err := s.client.CreateSecretInNamespace(dc.EnvironmentName(), secret); err != nil {
		return err
	}
	return nil
}

// CreateBuildConfig creates a buildconfig in the OSCP
func (s Service) CreateBuildConfig(dc controller.DeployCmd, description, fromNamespace, fromImageName string) error {
	bc := &bc.BuildConfig{
		ObjectMeta: api.ObjectMeta{
			Name:   dc.AppName(),
			Labels: dc.Labels(),
			Annotations: map[string]string{
				"rhmap/description": description,
				"rhmap/title":       dc.AppName(),
				"description":       "buildconfig for " + dc.AppName(),
			},
		},
		Spec: bc.BuildConfigSpec{
			RunPolicy: bc.BuildRunPolicySerialLatestOnly,
			CommonSpec: bc.CommonSpec{
				Source: bc.BuildSource{
					Git: &bc.GitBuildSource{
						URI: dc.SourceLoc(),
						Ref: dc.SourceBranch(),
					},
					SourceSecret: &api.LocalObjectReference{
						Name: dc.AppName(),
					},
				},
				Strategy: bc.BuildStrategy{
					SourceStrategy: &bc.SourceBuildStrategy{
						From: api.ObjectReference{
							Kind:      "ImageStreamTag",
							Namespace: fromNamespace,
							Name:      fromImageName,
						},
						Env: formatEnvVars(dc.GetEnvVars()),
					},
				},
				Output: bc.BuildOutput{
					To: &api.ObjectReference{
						Kind: "ImageStreamTag",
						Name: dc.AppName() + ":latest",
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

	if _, err := s.client.CreateBuildConfigInNamespace(dc.EnvironmentName(), bc); err != nil {
		return err
	}

	return nil
}

// CreateDeploymentConfig creates a deploymentconfig in the OSCP
func (s Service) CreateDeploymentConfig(dCmd controller.DeployCmd, fromImage, description string) error {
	dc := &dcapi.DeploymentConfig{
		ObjectMeta: api.ObjectMeta{
			Name:   dCmd.AppName(),
			Labels: dCmd.Labels(),
			Annotations: map[string]string{
				"rhmap/description": description,
				"rhmap/title":       dCmd.AppName(),
				"description":       "Deploy config for " + dCmd.AppName(),
			},
		},
		Spec: dcapi.DeploymentConfigSpec{
			Strategy: dcapi.DeploymentStrategy{
				Type: dcapi.DeploymentStrategyTypeRolling,
			},
			Triggers: []dcapi.DeploymentTriggerPolicy{
				{
					Type: dcapi.DeploymentTriggerOnImageChange,
					ImageChangeParams: &dcapi.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							dCmd.AppName(),
						},
						From: api.ObjectReference{
							Kind: "ImageStream",
							Name: dCmd.AppName(),
						},
					},
				},
				{
					Type: dcapi.DeploymentTriggerOnConfigChange,
				},
			},
			Replicas: 1,
			Selector: map[string]string{
				"name": dCmd.AppName(),
			},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Name: dCmd.AppName(),
					Labels: map[string]string{
						"name": dCmd.AppName(),
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  dCmd.AppName(),
							Image: fromImage,
							Ports: []api.ContainerPort{
								{
									ContainerPort: 8001,
								},
							},
							Env: formatEnvVars(dCmd.GetEnvVars()),
							Resources: api.ResourceRequirements{
								Limits: api.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("250Mi"),
								},
								Requests: api.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("90Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := s.client.CreateDeployConfigInNamespace(dCmd.EnvironmentName(), dc); err != nil {
		return err
	}

	return nil
}
