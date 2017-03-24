package status

import (
	"encoding/json"

	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

// ErrNotExist means the status does not exist
type ErrNotExist string

func (ee ErrNotExist) Error() string {
	return "error status does not exist"
}

// RedisRetrieverPublisher implements the StatusPublisher and StatusRetriever interface wrapping around redis
type RedisRetrieverPublisher struct {
	*redis.Client
}

// New return a new RedisRetrieverPublisher
func New(client *redis.Client) *RedisRetrieverPublisher {
	return &RedisRetrieverPublisher{
		Client: client,
	}
}

// Get will retrieve a given status by key
func (rp *RedisRetrieverPublisher) Get(key string) (*deploy.ConfigurationStatus, error) {
	var ret = deploy.ConfigurationStatus{}
	val, err := rp.Client.Get(key).Result()
	if err != nil {
		return nil, errors.Wrap(err, "failed to Get key "+key+" in RedisRetreiverPublisher")
	}
	if err := json.Unmarshal([]byte(val), &ret); err != nil {
		return nil, errors.Wrap(err, "failed to decode ConfigurationStatus ")
	}
	return &ret, nil
}

// Publish will update a given status atomically
func (rp *RedisRetrieverPublisher) Publish(key string, status deploy.ConfigurationStatus) error {
	if _, err := rp.GetSet(key, status).Result(); err != nil {
		return errors.Wrap(err, "failed to publish status update")
	}
	return nil
}
