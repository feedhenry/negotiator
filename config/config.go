package config

import (
	"os"

	"github.com/pkg/errors"
)

type Conf struct {
}

func (c *Conf) Namespace(override string) string {
	if override != "" {
		return override
	}
	//populated by the downward api https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/downward-api.md
	return os.Getenv("DEPLOY_NAMESPACE")
}

func (c *Conf) APIHost() string {
	return os.Getenv("API_HOST")
}

func (c *Conf) APIToken() string {
	return os.Getenv("API_TOKEN")
}

func (c Conf) RepoDir() string {
	return os.Getenv("REPO_DIR")
}

func (c Conf) Validate() error {
	var err string

	if "" == c.RepoDir() {
		err += " : Missing Needed Env Var REPO_DIR"
	}
	if err != "" {
		return errors.New(err)
	}
	return nil
}
