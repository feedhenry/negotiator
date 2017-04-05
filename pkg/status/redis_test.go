package status_test

import (
	"testing"

	"fmt"

	"github.com/feedhenry/negotiator/pkg/config"
	"github.com/feedhenry/negotiator/pkg/status"
	redis "github.com/go-redis/redis"
)

// TestStatus relies on setting "REDIS_SERVICE_HOST" "REDIS_SERVICE_PORT" in the env can be skipped using -short flag
func TestStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test TestStatus")
	}
	cases := []struct {
		Name               string
		Status             string
		Description        string
		ExpectGetError     bool
		ExpectPublishError bool
		GetKey             string
		PubKey             string
		Assert             func(cs *status.Status) error
	}{
		{
			Name:               "test publish and get configuration status",
			Status:             "in progress",
			Description:        "a Description",
			ExpectGetError:     false,
			ExpectPublishError: false,
			PubKey:             "test",
			GetKey:             "test",
			Assert: func(cs *status.Status) error {
				if cs == nil {
					return fmt.Errorf("expected a ConfigurationStatus but got none")
				}
				return nil
			},
		},
		{
			Name:               "test get configuration status thats not there",
			Status:             "in progress",
			Description:        "a Description",
			ExpectGetError:     true,
			ExpectPublishError: false,
			PubKey:             "test",
			GetKey:             "notthere",
			Assert: func(cs *status.Status) error {
				if cs != nil {
					return fmt.Errorf("expected NOT to get a ConfigurationStatus but got one")
				}
				return nil
			},
		},
	}
	conf := &config.Conf{}

	redisOpts := conf.Redis()
	redisClient := redis.NewClient(&redisOpts)
	defer redisClient.Close()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ret := status.New(redisClient)
			err := ret.Publish(tc.PubKey, tc.Status, tc.Description)
			if tc.ExpectPublishError && err == nil {
				t.Fatalf("expected a publish error but got none ")
			}
			val, err := ret.Get(tc.GetKey)
			if tc.ExpectGetError && err == nil {
				t.Fatalf("expected a get error but got none")
			}
			if err := tc.Assert(val); err != nil {
				t.Fatalf("unexpected assert error %s ", err.Error())
			}
		})
	}

}
