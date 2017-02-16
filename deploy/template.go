package deploy

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

	roapi "github.com/openshift/origin/pkg/route/api"
)

type TemplateLoader interface {
	Load(name string) (*template.Template, error)
}

type TemplateDecoder interface {
	Decode(template []byte) (*api.Template, error)
}

// PaaSClient is the interface this controller expects for interacting with an openshift paas
type PaaSClient interface {
	ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error)
	CreateServiceInNamespace(ns string, svc *k8api.Service) (*k8api.Service, error)
	CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error)
	CreateImageStream(ns string, i *image.ImageStream) (*image.ImageStream, error)
	CreateBuildConfigInNamespace(namespace string, b *bc.BuildConfig) (*bc.BuildConfig, error)
	CreateDeployConfigInNamespace(namespace string, d *dc.DeploymentConfig) (*dc.DeploymentConfig, error)
	CreateSecretInNamespace(namespace string, s *k8api.Secret) (*k8api.Secret, error)
	InstantiateBuild(ns, buildName string) (*bc.Build, error)
}

// Controller handle deploy templates to OSCP
type Controller struct {
	templateLoader  TemplateLoader
	TemplateDecoder TemplateDecoder
	PaaSClient      PaaSClient
}

// Payload represents a deployment payload
type Payload struct {
	ServiceName  string    `json:"serviceName"`
	Route        string    `json:"route"`
	ProjectGuid  string    `json:"projectGuid"`
	CloudAppGuid string    `json:"cloudAppGuid"`
	Domain       string    `json:"domain"`
	Env          string    `json:"env"`
	Replicas     int       `json:"replicas"`
	EnvVars      []*EnvVar `json:"envVars"`
	Repo         *Repo     `json:"repo"`
	Target       *Target   `json:"target"`
}

type Target struct {
	Host  string `json:"host"`
	Token string `json:"token"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Repo represents a git repo
type Repo struct {
	Loc  string `json:"loc"`
	Ref  string `json:"ref"`
	Auth struct {
		AuthType string `json:"authType"` //basic or ssh
		User     string `json:"user"`
		Key      string `json:"key"`
	} `json:"auth"`
}

const templateCloudApp = "cloudapp"
const templateCache = "cache"

func (p Payload) Validate(template string) error {
	switch template {
	case templateCloudApp:
		if p.Repo == nil || p.Repo.Loc == "" || p.Repo.Ref == "" {
			return errors.New("a repo url and ref is required")
		}
	}
	if p.ServiceName == "" {
		return errors.New("a serviceName is required")
	}
	return nil
}

// New returns a new Controller
func New(tl TemplateLoader, td TemplateDecoder) *Controller {
	return &Controller{
		templateLoader:  tl,
		TemplateDecoder: td,
	}
}

// Template deploys a set of objects based on a template. Templates are located in resources/templates
func (c Controller) Template(client PaaSClient, template, nameSpace string, deploy *Payload) error {
	var (
		buf bytes.Buffer
	)
	tpl, err := c.templateLoader.Load(template)
	if err != nil {
		return errors.Wrap(err, "failed to load template "+template)
	}
	if err := deploy.Validate(template); err != nil {
		return err
	}
	if err := tpl.ExecuteTemplate(&buf, template, deploy); err != nil {
		return errors.Wrap(err, "failed to execute template")
	}
	osTemplate, err := c.TemplateDecoder.Decode(buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "failed to decode into a os template")
	}
	for _, ob := range osTemplate.Objects {
		switch ob.(type) {
		case *dc.DeploymentConfig:
			if _, err := client.CreateDeployConfigInNamespace(nameSpace, ob.(*dc.DeploymentConfig)); err != nil {
				return err
			}
		case *k8api.Service:
			if _, err := client.CreateServiceInNamespace(nameSpace, ob.(*k8api.Service)); err != nil {
				return err
			}
		case *route.Route:
			if _, err := client.CreateRouteInNamespace(nameSpace, ob.(*route.Route)); err != nil {
				return err
			}
		case *image.ImageStream:
			if _, err := client.CreateImageStream(nameSpace, ob.(*image.ImageStream)); err != nil {
				return err
			}
		case *bc.BuildConfig:
			if _, err := client.CreateBuildConfigInNamespace(nameSpace, ob.(*bc.BuildConfig)); err != nil {
				return err
			}
		}
	}
	//we only need to instantiate a build if it is cloud app
	if template != templateCloudApp {
		return nil
	}
	if _, err := client.InstantiateBuild(nameSpace, deploy.ServiceName); err != nil {
		return err
	}

	return nil
}
