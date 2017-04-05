package status

import "github.com/feedhenry/negotiator/pkg/log"

// MultiStatus will publish to both the log and redis
type MultiStatus struct {
	redis     *RedisRetrieverPublisher
	log       *LogStatusPublisher
	stdLogger log.Logger
}

// NewMultiStatus returns  a status publisher that publisher to both redis and the log
func NewMultiStatus(redisPublisher *RedisRetrieverPublisher, logPublisher *LogStatusPublisher, logger log.Logger) *MultiStatus {
	return &MultiStatus{
		redis:     redisPublisher,
		log:       logPublisher,
		stdLogger: logger,
	}
}

// Publish will update a given status
func (rp *MultiStatus) Publish(key string, status, description string) error {
	if err := rp.log.Publish(key, status, description); err != nil {
		rp.stdLogger.Error("failed to publish status to log " + err.Error())
	}
	if err := rp.redis.Publish(key, status, description); err != nil {
		rp.stdLogger.Error("failed to publish status to redis  " + err.Error())
	}
	return nil
}

func (rp *MultiStatus) Clear(key string) error {
	return rp.redis.Clear(key)
}
