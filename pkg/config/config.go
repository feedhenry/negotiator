package config

import (
	"os"

	"github.com/go-redis/redis"

	"fmt"

	"github.com/pkg/errors"
)

type Conf struct {
}

func (c Conf) TemplateDir() string {
	return os.Getenv("TEMPLATE_DIR")
}

func (c Conf) Validate() error {
	var err string

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
