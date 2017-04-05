package status

import (
	"encoding/json"

	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

// ErrNotExist means the status does not exist
type ErrStatusNotExist struct {
	statusKey string
}

func (ene *ErrStatusNotExist) Error() string {
	return "status not found for key " + ene.statusKey
}

func IsErrStatusNotExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ErrStatusNotExist)
	return ok
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

// Status represent the current status of the configuration
type Status struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Log         []string  `json:"log"`
	Started     time.Time `json:"-"`
}

// Get will retrieve a given status by key
func (rp *RedisRetrieverPublisher) Get(key string) (*Status, error) {
	var ret = Status{}
	val, err := rp.Client.Get(key).Result()
	if err == redis.Nil {
		return nil, &ErrStatusNotExist{statusKey: key}
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to Get key "+key+" in RedisRetreiverPublisher")
	}
	if err := json.Unmarshal([]byte(val), &ret); err != nil {
		return nil, errors.Wrap(err, "failed to decode ConfigurationStatus ")
	}
	return &ret, nil
}

// Clear removes the key from redis
func (rp *RedisRetrieverPublisher) Clear(key string) error {
	return rp.Client.Del(key).Err()
}

// Publish will update a given status
func (rp *RedisRetrieverPublisher) Publish(key string, status, description string) error {
	val, err := rp.Get(key)
	if IsErrStatusNotExists(err) {
		val = &Status{}
	} else if err != nil {
		return errors.Wrap(err, "unexpected error in Publish ")
	}
	val.Description = description
	val.Log = append(val.Log, description)
	val.Status = status
	data, err := json.Marshal(val)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal status in RedisRetreiverPublisher")
	}
	return rp.Client.Set(key, string(data), time.Minute*20).Err()
}
