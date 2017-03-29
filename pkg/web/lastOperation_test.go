package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"

	"io/ioutil"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/status"
	"github.com/feedhenry/negotiator/pkg/web"
)

type mockStatusRetriever struct {
	status map[string]*deploy.Status
	err    error
}

func (msr mockStatusRetriever) Get(key string) (*deploy.Status, error) {
	return msr.status[key], msr.err
}

func setUpLastOperationHandler(statusRetriever web.StatusRetriever) http.Handler {
	router := web.BuildRouter()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logger := logrus.StandardLogger()
	web.LastOperation(router, statusRetriever, logger)
	return web.BuildHTTPHandler(router)
}

//GET /v2/service_instances/:instance_id/last_operation
func TestGetLastOperation(t *testing.T) {
	cases := []struct {
		Name            string
		StatusRetriever mockStatusRetriever
		Operation       string
		InstanceID      string
		StatusCode      int
		AssertStatus    func(cs *deploy.Status) error
	}{
		{
			StatusCode: 200,
			Name:       "test get last action happy",
			Operation:  "provision",
			InstanceID: "test",
			AssertStatus: func(cs *deploy.Status) error {
				if cs == nil {
					return fmt.Errorf("expected a ConfigurationStatus but got none")
				}
				return nil
			},
			StatusRetriever: mockStatusRetriever{
				status: map[string]*deploy.Status{
					"test:provision": &deploy.Status{
						Status:      "success",
						Description: "completed setup",
						Log:         []string{},
					},
				},
			},
		},
		{
			StatusCode: 404,
			Name:       "test get last action not found",
			Operation:  "provision",
			InstanceID: "test",
			AssertStatus: func(cs *deploy.Status) error {
				if cs != nil {
					return fmt.Errorf("expected no ConfigurationStatus but got one")
				}
				return nil
			},
			StatusRetriever: mockStatusRetriever{
				status: map[string]*deploy.Status{
					"something:provision": &deploy.Status{
						Status:      "success",
						Description: "completed setup",
						Log:         []string{},
					},
				},
				err: &status.ErrStatusNotExist{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			handler := setUpLastOperationHandler(tc.StatusRetriever)
			s := httptest.NewServer(handler)
			defer s.Close()
			url := fmt.Sprintf("%s/v2/service_instances/%s/last_operation?operation=%s", s.URL, tc.InstanceID, tc.Operation)
			res, err := http.Get(url)
			if err != nil {
				t.Fatalf("failed to make request to last_operation handler %s ", err.Error())
			}
			defer res.Body.Close()
			if tc.StatusCode != res.StatusCode {
				t.Fatalf("expected code %v but got %v", tc.StatusCode, res.StatusCode)
			}
			if res.StatusCode != 200 {
				// no need to parse data if not 200
				return
			}
			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("reading response from last_operation handler failed %s ", err.Error())
			}
			status := &deploy.Status{}
			if err := json.Unmarshal(data, status); err != nil {
				t.Fatalf("unexpeted error during Unmarshal %s", err.Error())
			}
			if err := tc.AssertStatus(status); err != nil {
				t.Fatalf("error asserting ConfigurationStatus %s ", err.Error())
			}
		})
	}

}
