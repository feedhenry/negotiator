package config

import (
	"os"

	"github.com/go-redis/redis"

	"fmt"

	"github.com/pkg/errors"
	"strconv"
)

type Conf struct {
}

func (c Conf) TemplateDir() string {
	return os.Getenv("TEMPLATE_DIR")
}

func (c Conf) DependencyTimeout() int {
	// error already handled in the Validate method
	val, _ := strconv.Atoi(os.Getenv("DEPENDENCY_TIMEOUT"))

	return val
}

func (c Conf) Validate() error {
	var err string

	dependencyTimeout, error := strconv.Atoi(os.Getenv("DEPENDENCY_TIMEOUT"))
	if error != nil {
		err += " : env var DEPENDENCY_TIMEOUT must be numeric"
	}

	if dependencyTimeout <= 0 {
		err += " : env var DEPENCY_TIMEOUT must be a positive number"
	}

	if "" == c.TemplateDir() {
		err += " : Missing Needed Env Var REPO_DIR"
	}
	if err != "" {
		return errors.New(err)
	}
	return nil
}

func (c Conf) Redis() redis.Options {
	addr := fmt.Sprintf("%s:%s", os.Getenv("REDIS_SERVICE_HOST"), os.Getenv("REDIS_SERVICE_PORT"))
	return redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_SERVICE_PASS"),
		DB:       0,
	}
}
