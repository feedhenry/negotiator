package status

import "github.com/feedhenry/negotiator/pkg/log"

// LogStatusPublisher publishes the status to the log
type LogStatusPublisher struct {
	Logger log.Logger
}

// Publish is called to send something new to the log
func (lsp LogStatusPublisher) Publish(key string, status, description string) error {
	lsp.Logger.Info(key, status, description)
	return nil
}

// Clear is a dummy implementation
func (lsp LogStatusPublisher) Clear(key string) error {
	return nil
}
