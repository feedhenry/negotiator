package deploy_test

import "testing"

type mockStatusPublisher struct {
	Statuses []string
}

func (msp *mockStatusPublisher) Publish(key string, status, description string) error {
	msp.Statuses = append(msp.Statuses, status)
	return nil
}

func (msp *mockStatusPublisher) Clear(key string) error {
	return nil
}

func TestConfigure(t *testing.T) {
	t.Skip("STILL NEED TO WRITE THIS TEST ")
}
