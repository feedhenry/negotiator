package deploy

//deploy is how rhmap handles deploys of services to OpenShift

import (
	"bytes"

	"text/template"

	bc "github.com/openshift/origin/pkg/build/api"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
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
	ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error)
	CreateServiceInNamespace(ns string, svc *k8api.Service) (*k8api.Service, error)
	CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error)
	CreateImageStream(ns string, i *image.ImageStream) (*image.ImageStream, error)
	CreateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error)
	CreateDeployConfigInNamespace(namespace string, d *dc.DeploymentConfig) (*dc.DeploymentConfig, error)
	CreateSecretInNamespace(namespace string, s *k8api.Secret) (*k8api.Secret, error)
	InstantiateBuild(ns, buildName string) (*bc.Build, error)
	FindDeploymentConfigByLabel(ns string, searchLabels map[string]string) (*dc.DeploymentConfig, error)
	DeployLogURL(ns, dc string) string
	BuildConfigLogURL(ns, dc string) string
	BuildURL(ns, bc, id string) string
}

// Controller handle deploy templates to OSCP
type Controller struct {
	templateLoader  TemplateLoader
	TemplateDecoder TemplateDecoder
	Logger          log.Logger
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

// Complete represents what is returned when a deploy completes sucessfully
type Complete struct {
	WatchURL string       `json:"watchURL"`
	Route    *roapi.Route `json:"route"`
	BuildURL string       `json:"buildURL"`
}

// Target is part of a Payload to deploy it is the target OSCP
type Target struct {
	Host  string `json:"host"`
	Token string `json:"token"`
}

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
	if p.ServiceName == "" {
		return ErrInvalid{message: "a serviceName is required"}
	}
	return nil
}

// New returns a new Controller
func New(tl TemplateLoader, td TemplateDecoder, logger log.Logger) *Controller {
	return &Controller{
		templateLoader:  tl,
		TemplateDecoder: td,
		Logger:          logger,
	}
}

// Template deploys a set of objects based on a template. Templates are located in resources/templates
func (c Controller) Template(client Client, template, nameSpace string, deploy *Payload) (*Complete, error) {
	var (
		buf  bytes.Buffer
		comp *Complete
	)
	if nameSpace == "" {
		return nil, errors.New("an empty namespace cannot be provided")
	}
	if err := deploy.Validate(template); err != nil {
		return nil, err
	}
	tpl, err := c.templateLoader.Load(template)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load template "+template)
	}
	if err := tpl.ExecuteTemplate(&buf, template, deploy); err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}
	osTemplate, err := c.TemplateDecoder.Decode(buf.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode into a os template")
	}
	dc, err := client.FindDeploymentConfigByLabel(nameSpace, map[string]string{"rhmap/guid": deploy.CloudAppGUID, "rhmap/title": deploy.ServiceName})
	if err != nil {
		return nil, errors.Wrap(err, "error trying to find deployment config")
	}
	if nil == dc {
		comp, err = c.create(client, osTemplate, nameSpace, deploy)
		if err != nil {
			return nil, err
		}
	} else {
		comp, err = c.update(client, dc, osTemplate, nameSpace, deploy)
		if err != nil {
			return nil, err
		}
	}
	//we only need to instantiate a build if it is cloud app
	if template != templateCloudApp {
		comp.WatchURL = client.DeployLogURL(nameSpace, deploy.ServiceName)
		return comp, nil
	}
	build, err := client.InstantiateBuild(nameSpace, deploy.ServiceName)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, errors.New("no build returned from call to OSCP. Unable to continue")
	}
	comp.WatchURL = client.BuildConfigLogURL(nameSpace, build.Name)
	comp.BuildURL = client.BuildURL(nameSpace, build.Name, deploy.CloudAppGUID)
	return comp, nil
}

func (c Controller) create(client Client, template *Template, nameSpace string, deploy *Payload) (*Complete, error) {
	var (
		complete = &Complete{}
	)
	for _, ob := range template.Objects {
		switch ob.(type) {
		case *dc.DeploymentConfig:
			_, err := client.CreateDeployConfigInNamespace(nameSpace, ob.(*dc.DeploymentConfig))
			if err != nil {
				return nil, err
			}
		case *k8api.Service:
			if _, err := client.CreateServiceInNamespace(nameSpace, ob.(*k8api.Service)); err != nil {
				return nil, err
			}
		case *route.Route:
			r, err := client.CreateRouteInNamespace(nameSpace, ob.(*route.Route))
			if err != nil {
				return nil, err
			}
			complete.Route = r
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
	return complete, nil
}

func (c Controller) update(client Client, dc *dc.DeploymentConfig, template *Template, nameSpace string, deploy *Payload) (*Complete, error) {
	var (
		complete = &Complete{}
	)
	// for _, c := range dc.Spec.Template.Spec.Containers{
	// 	c.Env
	// }
	//update git details
	// update env vars
	return complete, errors.New("redeploy not implemented")
}
