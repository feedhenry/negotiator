package controller

// PaaSService defines what the handler expects from a service interacting with the PAAS
type PaaSService interface {
	CreateService(dc DeployCmd, description string, port int32) error
	CreateRoute(dc DeployCmd, optionalHost string) error
	CreateImageStream(dc DeployCmd) error
	CreateSecret(dc DeployCmd, data map[string][]byte) error
	CreateBuildConfig(dc DeployCmd, description, fromNamespace, fromImageName string) error
	CreateDeploymentConfig(dc DeployCmd, fromImage, description string) error
}

// EnvVar defines an environment variable
type EnvVar struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Deploy struct {
	paaSService PaaSService
}

func NewDeployController(paaSService PaaSService) Deploy {
	return Deploy{
		paaSService: paaSService,
	}
}

// DeployCmd encapsulates what data is required to do a deploy
type DeployCmd interface {
	SetAppTag(string)
	GetAppTag() string
	Labels() map[string]string
	AppName() string
	EnvironmentName() string
	CloudAppGUID() string
	Project() string
	DomainName() string
	UserName() string
	Authentication() string
	SourceLoc() string
	SourceBranch() string
	GetEnvVars() []*EnvVar
	AddEnvVar(name, value string)
}

type DeployResponse struct {
	Status string `json:"status"`
	LogURL string `json:"logURL"`
}

func (d Deploy) Run(dc DeployCmd) (interface{}, error) {
	exists, err := d.buildConfigExists(dc.CloudAppGUID())
	if err != nil {
		return nil, err
	}
	if exists {
		return d.update(dc)
	}
	return d.create(dc)
}

func (d Deploy) buildConfigExists(name string) (bool, error) {
	return false, nil
}

func (d Deploy) create(dc DeployCmd) (*DeployResponse, error) {
	appTag := dc.GetAppTag()
	if appTag == "" {
		appTag = "cloud"
	}

	dc.SetAppTag("redis")
	d.createRedis(dc)
	dc.AddEnvVar("REDIS_HOST", dc.AppName())
	dc.SetAppTag(appTag)
	return d.createCloudApp(dc)
}

func (d Deploy) createRedis(dc DeployCmd) error {
	if err := d.paaSService.CreateService(dc, "redis app", 8001); err != nil {
		return err
	}
	if err := d.paaSService.CreateDeploymentConfig(dc, "docker.io/rhmap/redis:2.18.22", "redis deployment config"); err != nil {
		return err
	}
	return nil
}

func (d Deploy) createCloudApp(dc DeployCmd) (*DeployResponse, error) {
	if err := d.paaSService.CreateService(dc, "rhmap cloud app", 8001); err != nil {
		return nil, err
	}
	if err := d.paaSService.CreateRoute(dc, ""); err != nil {
		return nil, err
	}
	if err := d.paaSService.CreateImageStream(dc); err != nil {
		return nil, err
	}
	//create secrets
	if err := d.paaSService.CreateSecret(dc, map[string][]byte{
		"username": []byte(dc.UserName()),
		"password": []byte(dc.Authentication()),
	}); err != nil {
		return nil, err
	}

	//create build config
	if err := d.paaSService.CreateBuildConfig(dc, "Build Config for RHMAP cloud app", "openshift", "nodejs:4"); err != nil {
		return nil, err
	}

	if err := d.paaSService.CreateDeploymentConfig(dc, dc.AppName() + ":latest", "rhmap cloud app"); err != nil {
		return nil, err
	}

	// TODO change to actual log URL
	return &DeployResponse{Status: "inprogress", LogURL: "http://mybuildlogurl.com/url"}, nil
}

func (d Deploy) update(dc DeployCmd) (DeployResponse, error) {
	//update logic
	return DeployResponse{}, nil
}
