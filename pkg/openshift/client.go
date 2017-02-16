package openshift

import (
	"flag"
	"os"

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
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	clientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
	kubectlutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
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

// DefaultClient will return a sane default client configured for the given host and token
func DefaultClient(host, token string) (Client, error) {
	defaultConfig := BuildDefaultConfig(host, token)
	return ClientFromConfig(defaultConfig)
}

// ClientFromConfig returns a client that wraps both kubernetes and openshift operations
func ClientFromConfig(conf clientcmd.ClientConfig) (Client, error) {
	factory := kubectlutil.NewFactory(conf)
	var oc *oclient.Client
	factory.BindFlags(flags)
	factory.BindExternalFlags(flags)
	if err := flags.Parse(os.Args); err != nil {
		return Client{}, errors.Wrap(err, "failed parsing flags")
	}
	flag.CommandLine.Parse([]string{})
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
		k8: kubeClient,
		oc: oc,
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
	k8 *k8client.Client
	oc *oclient.Client
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
	return build, err
}

func (c Client) CreateServiceInNamespace(ns string, svc *api.Service) (*api.Service, error) {
	s, err := c.k8.Services(ns).Create(svc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create service")
	}
	return s, err
}

func (c Client) CreateSecretInNamespace(ns string, s *api.Secret) (*api.Secret, error) {
	s, err := c.k8.Secrets(ns).Create(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create service")
	}
	return s, err
}

func (c Client) CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	route, err := c.oc.Routes(ns).Create(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create route")
	}
	return route, err
}

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
