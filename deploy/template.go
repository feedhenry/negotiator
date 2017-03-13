package deploy

//deploy is how rhmap handles deploys of services to OpenShift

import (
	"bytes"

	"text/template"

	bc "github.com/openshift/origin/pkg/build/api"
	dc "github.com/openshift/origin/pkg/deploy/api"
	image "github.com/openshift/origin/pkg/image/api"

	route "github.com/openshift/origin/pkg/route/api"
	"github.com/openshift/origin/pkg/template/api"
	"github.com/pkg/errors"
	k8api "k8s.io/kubernetes/pkg/api"

	"github.com/feedhenry/negotiator/pkg/log"
	roapi "github.com/openshift/origin/pkg/route/api"
)

// TemplateLoader defines how deploy wants to load templates in order to be able to deploy them.
type TemplateLoader interface {
	Load(name string) (*template.Template, error)
}

// TemplateDecoder defines how deploy wants to decode the templates into data structures
type TemplateDecoder interface {
	Decode(template []byte) (*Template, error)
}

// Client is the interface this controller expects for interacting with an openshift paas
type Client interface {
	CreateServiceInNamespace(ns string, svc *k8api.Service) (*k8api.Service, error)
	CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error)
	CreateImageStream(ns string, i *image.ImageStream) (*image.ImageStream, error)
	CreateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error)
	CreateDeployConfigInNamespace(namespace string, d *dc.DeploymentConfig) (*dc.DeploymentConfig, error)
	CreateSecretInNamespace(namespace string, s *k8api.Secret) (*k8api.Secret, error)
	UpdateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error)
	UpdateDeployConfigInNamespace(ns string, d *dc.DeploymentConfig) (*dc.DeploymentConfig, error)
	UpdateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error)
	InstantiateBuild(ns, buildName string) (*bc.Build, error)
	FindDeploymentConfigsByLabel(ns string, searchLabels map[string]string) ([]dc.DeploymentConfig, error)
	FindBuildConfigByLabel(ns string, searchLabels map[string]string) (*bc.BuildConfig, error)
	GetDeploymentConfigByName(ns, deploymentName string) (*dc.DeploymentConfig, error)
	DeployLogURL(ns, dc string) string
	BuildConfigLogURL(ns, dc string) string
	BuildURL(ns, bc, id string) string
}

// Controller handle deploy templates to OSCP
type Controller struct {
	templateLoader          TemplateLoader
	TemplateDecoder         TemplateDecoder
	Logger                  log.Logger
	ConfigurationController *EnvironmentServiceConfigController
}

// Payload represents a deployment payload
type Payload struct {
	ServiceName  string    `json:"serviceName"`
	Route        string    `json:"route"`
	ProjectGUID  string    `json:"projectGuid"`
	CloudAppGUID string    `json:"cloudAppGuid"`
	Domain       string    `json:"domain"`
	Env          string    `json:"env"`
	Replicas     int       `json:"replicas"`
	EnvVars      []*EnvVar `json:"envVars"`
	Repo         *Repo     `json:"repo"`
	Target       *Target   `json:"target"`
}

// Dispatched represents what is returned when a deploy dispatches sucessfully
type Dispatched struct {
	// WatchURL is the url used to watch and stream the logs of a deployment or build
	WatchURL string       `json:"watchURL"`
	Route    *roapi.Route `json:"route"`
	// BuildURL is the url used to get the status of a build
	BuildURL string `json:"buildURL"`

	DepoymentName string `json:"deploymentName"`
}

// Target is part of a Payload to deploy it is the target OSCP
type Target struct {
	Host  string `json:"host"`
	Token string `json:"token"`
}

// EnvVar defines an environment variables name and value
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Repo represents a git Repo
type Repo struct {
	Loc  string `json:"loc"`
	Ref  string `json:"ref"`
	Auth struct {
		AuthType string `json:"authType"` //basic or ssh
		User     string `json:"user"`
		Key      string `json:"key"`
	} `json:"auth"`
}

// Template wraps the OpenShift template to give us some domain specific logic
type Template struct {
	*api.Template
}

func (t Template) hasBuildConfig() bool {
	for _, o := range t.Objects {
		if _, ok := o.(*bc.BuildConfig); ok {
			return true
		}
	}
	return false
}

func (t Template) hasSecret() bool {
	for _, o := range t.Objects {
		if _, ok := o.(*k8api.Secret); ok {
			return true
		}
	}
	return false
}

const templateCloudApp = "cloudapp"
const templateCache = "cache"

type environmentServices []string

func (es environmentServices) isEnvironmentService(name string) bool {
	for _, k := range es {
		if k == name {
			return true
		}
	}
	return false
}

var availableEnvironmentServices = environmentServices{templateCache}

// ErrInvalid is returned when something invalid happens
type ErrInvalid struct {
	message string
}

func (e ErrInvalid) Error() string {
	return e.message
}

// Validate validates a payload
func (p Payload) Validate(template string) error {
	switch template {
	case templateCloudApp:
		if p.Repo == nil || p.Repo.Loc == "" || p.Repo.Ref == "" {
			return ErrInvalid{message: "a repo is expected for a cloudapp"}
		}
	}
	if p.Target == nil {
		return ErrInvalid{message: "an oscp target is required"}
	}
	if p.ServiceName == "" {
		return ErrInvalid{message: "a serviceName is required"}
	}

	return nil
}

type ServiceConfigFactory interface {
	Factory(serviceName string) Configurer
	Publisher(publisher StatusPublisher)
}

// New returns a new Controller
func New(tl TemplateLoader, td TemplateDecoder, logger log.Logger, configuration *EnvironmentServiceConfigController) *Controller {
	return &Controller{
		templateLoader:          tl,
		TemplateDecoder:         td,
		Logger:                  logger,
		ConfigurationController: configuration,
	}
}

// Template deploys a set of objects based on a template. Templates are located in resources/templates
func (c Controller) Template(client Client, template, nameSpace string, deploy *Payload) (*Dispatched, error) {
	var (
		buf        bytes.Buffer
		dispatched *Dispatched
	)
	// wrap up the logic for instansiating a build or not
	instansiateBuild := func() (*Dispatched, error) {

		if template != templateCloudApp {
			dispatched.WatchURL = client.DeployLogURL(nameSpace, deploy.ServiceName)
			return dispatched, nil
		}

		build, err := client.InstantiateBuild(nameSpace, deploy.ServiceName)
		if err != nil {
			return nil, err
		}
		if build == nil {
			return nil, errors.New("no build returned from call to OSCP. Unable to continue")
		}
		dispatched.WatchURL = client.BuildConfigLogURL(nameSpace, build.Name)
		dispatched.BuildURL = client.BuildURL(nameSpace, build.Name, deploy.CloudAppGUID)
		return dispatched, nil
	}

	if nameSpace == "" {
		return nil, errors.New("an empty namespace cannot be provided")
	}
	if err := deploy.Validate(template); err != nil {
		return nil, err
	}
	tpl, err := c.templateLoader.Load(template)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load template "+template+": ")
	}
	if err := tpl.ExecuteTemplate(&buf, template, deploy); err != nil {
		return nil, errors.Wrap(err, "failed to execute template: ")
	}
	osTemplate, err := c.TemplateDecoder.Decode(buf.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode into a os template: ")
	}
	searchCrit := map[string]string{"rhmap/guid": deploy.CloudAppGUID}
	dcs, err := client.FindDeploymentConfigsByLabel(nameSpace, searchCrit)
	if err != nil {
		return nil, errors.Wrap(err, "error trying to find deployment config: ")
	}
	bc, err := client.FindBuildConfigByLabel(nameSpace, searchCrit)
	if err != nil {
		return nil, errors.Wrap(err, "error trying to find build config: ")
	}
	//check if already deployed
	if len(dcs) > 0 || nil != bc {
		dispatched, err = c.update(client, &dcs[0], bc, osTemplate, nameSpace, deploy)
		if err != nil {
			return nil, errors.Wrap(err, "Error updating deploy: ")
		}
		c.ConfigurationController.Configure(client, dispatched.DepoymentName, nameSpace)
		return instansiateBuild()
	}
	dispatched, err = c.create(client, osTemplate, nameSpace, deploy)
	if err != nil {
		return nil, err
	}
	c.ConfigurationController.Configure(client, dispatched.DepoymentName, nameSpace)
	return instansiateBuild()

}

// create is responsible for creating the different Objects in a template via the OSCP and kubernetes API. this is used for new deployments
func (c Controller) create(client Client, template *Template, nameSpace string, deploy *Payload) (*Dispatched, error) {
	var (
		dispatched = &Dispatched{}
	)
	for _, ob := range template.Objects {
		switch ob.(type) {
		case *dc.DeploymentConfig:
			deployment := ob.(*dc.DeploymentConfig)
			deployed, err := client.CreateDeployConfigInNamespace(nameSpace, deployment)
			if err != nil {
				return nil, err
			}
			dispatched.DepoymentName = deployed.GetName()
		case *k8api.Service:
			if _, err := client.CreateServiceInNamespace(nameSpace, ob.(*k8api.Service)); err != nil {
				return nil, err
			}
		case *route.Route:
			r, err := client.CreateRouteInNamespace(nameSpace, ob.(*route.Route))
			if err != nil {
				return nil, err
			}
			dispatched.Route = r
		case *image.ImageStream:
			if _, err := client.CreateImageStream(nameSpace, ob.(*image.ImageStream)); err != nil {
				return nil, err
			}
		case *bc.BuildConfig:
			bConfig := ob.(*bc.BuildConfig)
			if _, err := client.CreateBuildConfigInNamespace(nameSpace, bConfig); err != nil {
				return nil, err
			}
		case *k8api.Secret:
			if _, err := client.CreateSecretInNamespace(nameSpace, ob.(*k8api.Secret)); err != nil {
				return nil, err
			}
		}
	}
	return dispatched, nil
}

// update the existing deployconfig and buildconfig with the new deployment payload data. Update the deployconfig (for env var updates),Update the buildconfig (for git repo and ref changes), Update any routes for the app
func (c Controller) update(client Client, d *dc.DeploymentConfig, b *bc.BuildConfig, template *Template, nameSpace string, deploy *Payload) (*Dispatched, error) {
	var (
		dispatched = &Dispatched{}
	)
	for _, ob := range template.Objects {
		switch ob.(type) {
		case *dc.DeploymentConfig:
			deployment := ob.(*dc.DeploymentConfig)
			deployment.SetResourceVersion(d.GetResourceVersion())
			deployed, err := client.UpdateDeployConfigInNamespace(nameSpace, deployment)
			if err != nil {
				return nil, errors.Wrap(err, "error updating deploy config: ")
			}
			dispatched.DepoymentName = deployed.GetName()
		case *bc.BuildConfig:
			ob.(*bc.BuildConfig).SetResourceVersion(b.GetResourceVersion())
			if _, err := client.UpdateBuildConfigInNamespace(nameSpace, ob.(*bc.BuildConfig)); err != nil {
				return nil, errors.Wrap(err, "error updating build config: ")
			}
		case *route.Route:
			r, err := client.UpdateRouteInNamespace(nameSpace, ob.(*route.Route))
			if err != nil {
				return nil, err
			}
			dispatched.Route = r
		}
	}
	return dispatched, nil
}
