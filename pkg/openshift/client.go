package openshift

import (
	"fmt"

	"github.com/feedhenry/negotiator/pkg/deploy"
	bc "github.com/openshift/origin/pkg/build/api"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
	oclient "github.com/openshift/origin/pkg/client"
	dc "github.com/openshift/origin/pkg/deploy/api"
	dcv1 "github.com/openshift/origin/pkg/deploy/api/v1"
	ioapi "github.com/openshift/origin/pkg/image/api"
	ioapi1 "github.com/openshift/origin/pkg/image/api/v1"
	roapi "github.com/openshift/origin/pkg/route/api"
	roapi1 "github.com/openshift/origin/pkg/route/api/v1"
	osapi "github.com/openshift/origin/pkg/template/api"
	osapi1 "github.com/openshift/origin/pkg/template/api/v1"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api"
	k8error "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/apis/batch"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	clientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
	kubectlutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)
)

func init() {
	osapi.AddToScheme(api.Scheme)
	osapi1.AddToScheme(api.Scheme)
	v1.AddToScheme(api.Scheme)
	bc.AddToScheme(api.Scheme)
	bcv1.AddToScheme(api.Scheme)
	dc.AddToScheme(api.Scheme)
	dcv1.AddToScheme(api.Scheme)
	roapi.AddToScheme(api.Scheme)
	roapi1.AddToScheme(api.Scheme)
	ioapi.AddToScheme(api.Scheme)
	ioapi1.AddToScheme(api.Scheme)
}

// ClientFactory will create and return an openshift client
type ClientFactory struct{}

// DefaultDeployClient will return a sane default client configured for the given host and token
func (ClientFactory) DefaultDeployClient(host, token string) (deploy.Client, error) {
	defaultConfig := BuildDefaultConfig(host, token)
	return ClientFromConfig(defaultConfig)
}

// ClientFromConfig returns a client that wraps both kubernetes and openshift operations
func ClientFromConfig(conf clientcmd.ClientConfig) (Client, error) {
	factory := kubectlutil.NewFactory(conf)
	var oc *oclient.Client
	kubeClient, err := factory.Client()
	if err != nil {
		return Client{}, errors.Wrap(err, "failed getting a kubernetes client")
	}
	restClientConfig, err := factory.ClientConfig()
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to get a restconfig")
	}
	ocfg := *restClientConfig
	ocfg.APIPath = ""
	oc, err = oclient.New(&ocfg)
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to get new oc")
	}

	return Client{
		k8:   kubeClient,
		oc:   oc,
		host: ocfg.Host,
	}, nil
}

// BuildDefaultConfig setups a  kube config with the given host and token
func BuildDefaultConfig(host, token string) clientcmd.ClientConfig {
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Clusters["local"] = &clientcmdapi.Cluster{
		Server:                host,
		InsecureSkipTLSVerify: true,
	}
	kubeConfig.AuthInfos["deployer"] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	kubeConfig.Contexts["local"] = &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "deployer",
	}

	conf := clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{
		CurrentContext: "local",
	})

	return conf
}

// Client is an external type that wraps both kubernetes and oc
type Client struct {
	k8   *k8client.Client
	oc   *oclient.Client
	host string
}

func (c Client) PodLogs(ns, name string, lines int64) ([]byte, error) {
	res := c.k8.Pods(ns).GetLogs(name, &api.PodLogOptions{TailLines: &lines})
	result := res.Do()
	if err := result.Error(); err != nil {
		return nil, errors.Wrap(err, "failed to get pod logs")
	}
	return result.Raw()
}

// FindServiceByLabel finds services deployed to the namespace based on searchLabels
func (c Client) FindServiceByLabel(ns string, searchLabels map[string]string) ([]api.Service, error) {
	opts := api.ListOptions{LabelSelector: labels.SelectorFromSet(searchLabels)}
	svc, err := c.k8.Services(ns).List(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list services for labels ")
	}
	return svc.Items, nil
}

// FindRouteByName returns a route by name from the namespace
func (c Client) FindRouteByName(ns, name string) (*roapi.Route, error) {
	r, err := c.oc.Routes(ns).Get(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to FindRouteByName")
	}
	return r, nil
}

// FindConfigMapByName returns a config map  for name
func (c Client) FindConfigMapByName(ns, name string) (*api.ConfigMap, error) {
	cm, err := c.k8.ConfigMaps(ns).Get(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to FindConfigMapByName")
	}
	return cm, nil
}

// CreateConfigMap creates a ConfigMap in the given namespace
func (c Client) CreateConfigMap(ns string, cm *api.ConfigMap) (*api.ConfigMap, error) {
	cm, err := c.k8.ConfigMaps(ns).Create(cm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to CreateConfigMap")
	}
	return cm, nil
}

// UpdateConfigMap updates the ConfigMap in the given namespace
func (c Client) UpdateConfigMap(ns string, cm *api.ConfigMap) (*api.ConfigMap, error) {
	var updated *api.ConfigMap
	err := k8client.RetryOnConflict(k8client.DefaultRetry, func() error {
		uptodate, err := c.k8.ConfigMaps(ns).Get(cm.Name)
		if err != nil {
			return err
		}
		cm.SetResourceVersion(uptodate.GetResourceVersion())
		updated, err = c.k8.ConfigMaps(ns).Update(cm)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to CreateConfigMap")
	}
	return updated, nil
}

// FindJobByName return nil if not found or the Job object
func (c Client) FindJobByName(ns, name string) (*batch.Job, error) {
	j, err := c.k8.BatchClient.Jobs(ns).Get(name)
	if err != nil && k8error.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to FindJobByName")
	}
	return j, nil
}

// CreatePod creates a pod object and returns it
func (c Client) CreatePod(ns string, p *api.Pod) (*api.Pod, error) {
	p, err := c.k8.Pods(ns).Create(p)
	if err != nil {
		return nil, errors.Wrap(err, "failed to CreatePod")
	}
	return p, nil
}

// CreateJobToWatch creates a job and returns a watch on that Job
func (c Client) CreateJobToWatch(j *batch.Job, ns string) (watch.Interface, error) {
	j, err := c.k8.BatchClient.Jobs(ns).Create(j)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Job")
	}
	opts := api.ListOptions{LabelSelector: labels.SelectorFromSet(j.Labels)}
	w, err := c.k8.BatchClient.Jobs(ns).Watch(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a watch on Job")
	}
	return w, nil
}

// ListBuildConfigs will return all build configs for the given namespace
func (c Client) ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error) {
	bl, err := c.oc.BuildConfigs(ns).List(api.ListOptions{}) //TODO may want to expose this in the func call
	if err != nil {
		return nil, errors.Wrap(err, "failed to list build configs")
	}
	out, err := api.Scheme.ConvertToVersion(bl, c.oc.APIVersion())
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert to object version ")
	}
	if v1bl, ok := out.(*bcv1.BuildConfigList); ok {
		return v1bl, nil
	}
	return nil, errors.New("unable to case the returned type to a BuildConfigList")
}

// CreateBuildConfigInNamespace creates the supplied build config in the supplied namespace and returns the buildconfig, or any errors that occurred
func (c Client) CreateBuildConfigInNamespace(ns string, b *bc.BuildConfig) (*bc.BuildConfig, error) {
	buildConfig, err := c.oc.BuildConfigs(ns).Create(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create BuildConfig")
	}
	return buildConfig, err
}

// UpdateBuildConfigInNamespace updates the supplied build config in the supplied namespace and returns the buildconfig, or any errors that occurred
func (c Client) UpdateBuildConfigInNamespace(ns string, b *bc.BuildConfig) (*bc.BuildConfig, error) {
	buildConfig, err := c.oc.BuildConfigs(ns).Update(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create BuildConfig")
	}
	return buildConfig, err
}

// InstantiateBuild will kick off a build in OSCP
func (c Client) InstantiateBuild(ns, buildName string) (*bc.Build, error) {
	//{"kind":"BuildRequest","apiVersion":"v1","metadata":{"name":"cloudapp"}}
	build, err := c.oc.BuildConfigs(ns).Instantiate(&bc.BuildRequest{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "BuildRequest",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name: buildName,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create build request")
	}
	if build == nil {
		return nil, errors.New("no build returned? cannot continue")
	}
	return build, err
}

// CreateServiceInNamespace creates the specified service in the specified namespace
func (c Client) CreateServiceInNamespace(ns string, svc *api.Service) (*api.Service, error) {
	s, err := c.k8.Services(ns).Create(svc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create service")
	}
	return s, err
}

// CreateSecretInNamespace creates the specified secret in the specified namespace
func (c Client) CreateSecretInNamespace(ns string, s *api.Secret) (*api.Secret, error) {
	s, err := c.k8.Secrets(ns).Create(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create service")
	}
	return s, err
}

// CreateRouteInNamespace creates the specified route in the specified namespace
func (c Client) CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	route, err := c.oc.Routes(ns).Create(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create route")
	}
	return route, err
}

// UpdateRouteInNamespace updates the supplied route in the supplied namespace and returns the route and any error
func (c Client) UpdateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	route, err := c.oc.Routes(ns).Update(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update route")
	}
	return route, err
}

// CreateImageStream creates the specified imagestream in the specified namespace
func (c Client) CreateImageStream(ns string, i *ioapi.ImageStream) (*ioapi.ImageStream, error) {
	imageStream, err := c.oc.ImageStreams(ns).Create(i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ImageStream")
	}
	return imageStream, err
}

// CreateDeployConfigInNamespace creates the supplied deploy config in the supplied namespace and returns the deployconfig, or any errors that occurred
func (c Client) CreateDeployConfigInNamespace(ns string, d *dc.DeploymentConfig) (*dc.DeploymentConfig, error) {
	deployConfig, err := c.oc.DeploymentConfigs(ns).Create(d)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DeployConfig")
	}
	return deployConfig, err
}

// GetDeploymentConfigByName returns the DeploymentConfig for the name passed
func (c Client) GetDeploymentConfigByName(ns, deploymentName string) (*dc.DeploymentConfig, error) {
	uptodate, err := c.oc.DeploymentConfigs(ns).Get(deploymentName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get DeploymentConfig")
	}
	return uptodate, nil
}

// CreatePersistentVolumeClaim uses the k8s api to setup the given claim
func (c Client) CreatePersistentVolumeClaim(namespace string, claim *api.PersistentVolumeClaim) (*api.PersistentVolumeClaim, error) {
	pclaim, err := c.k8.PersistentVolumeClaims(namespace).Create(claim)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PersistentVolumeClaim")
	}
	return pclaim, nil

}

// UpdateDeployConfigInNamespace updates the supplied deploy config in the supplied namespace and returns the deployconfig and any errors that occurred
func (c Client) UpdateDeployConfigInNamespace(ns string, d *dc.DeploymentConfig) (*dc.DeploymentConfig, error) {
	var deployed *dc.DeploymentConfig
	err := k8client.RetryOnConflict(k8client.DefaultRetry, func() error {
		uptodate, err := c.oc.DeploymentConfigs(ns).Get(d.Name)
		if err != nil {
			return err
		}
		d.SetResourceVersion(uptodate.GetResourceVersion())
		deployed, err = c.oc.DeploymentConfigs(ns).Update(d)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed update UpdateDeployConfigInNamespace")
	}
	return deployed, nil
}

// FindDeploymentConfigsByLabel searches for a deployment config by a label selector
func (c Client) FindDeploymentConfigsByLabel(ns string, searchLabels map[string]string) ([]dc.DeploymentConfig, error) {
	l, err := c.oc.DeploymentConfigs(ns).List(api.ListOptions{LabelSelector: labels.SelectorFromSet(searchLabels)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list deployconfig")
	}
	if nil == l || len(l.Items) == 0 {
		return nil, nil
	}
	return l.Items, nil
}

// FindBuildConfigByLabel searches for a build config by a label selector
func (c Client) FindBuildConfigByLabel(ns string, searchLabels map[string]string) (*bc.BuildConfig, error) {
	l, err := c.oc.BuildConfigs(ns).List(api.ListOptions{LabelSelector: labels.SelectorFromSet(searchLabels)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list buildconfig")
	}
	if nil == l || len(l.Items) == 0 {
		return nil, nil
	}
	return &l.Items[0], nil
}

// DeployLogURL returns the URL that can be polled for the deploy log of the specified dc in the specified namespace
func (c Client) DeployLogURL(ns, dc string) string {
	return fmt.Sprintf("%s/oapi/v1/namespaces/%s/deploymentconfigs/%s/log?follow=true", c.host, ns, dc)
}

// BuildConfigLogURL returns the URL that can be polled for the build log of the specified dc in the specified namespace
func (c Client) BuildConfigLogURL(ns, bc string) string {
	return fmt.Sprintf("%s/oapi/v1/namespaces/%s/builds/%s/log?follow=true", c.host, ns, bc)
}

// BuildURL returns the URL that can be polled for the build log of the specified build id for the specified dc in the specified namespace
func (c Client) BuildURL(ns, bc, id string) string {
	return fmt.Sprintf("%s/oapi/v1/namespaces/%s/builds?fieldSelector=metadata.name=%s&watch=true", c.host, ns, bc)
}
