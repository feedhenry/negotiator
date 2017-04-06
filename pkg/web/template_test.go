package web_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/pkg/web"
)

func setMarkServicesHandler() http.Handler {
	router := web.BuildRouter()
	pc := mock.NewPassClient()
	templates := openshift.NewTemplateLoaderDecoder("")
	clientFactory := mock.ClientFactory{PassClient: pc}
	web.Templates(router, templates, clientFactory)
	return web.BuildHTTPHandler(router)
}

func TestMarkServices(t *testing.T) {
	handler := setMarkServicesHandler()
	s := httptest.NewServer(handler)
	defer s.Close()
	url := fmt.Sprintf("%s/service/templates?env=rhmap-poc-core-dev", s.URL)
	t.Log(url)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	// expect a failure (we dont have headers)
	res, err := client.Do(req)
	if res.StatusCode != 406 {
		t.Fatalf("iexpected code %v but got %v", 406, res.StatusCode)
	}

	// all should work now:we
	req.Header.Set("X-RHMAP-HOST", "somehost")
	req.Header.Set("X-RHMAP-TOKEN", "sometoken")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request to services handler %s ", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("expected code %v but got %v", 200, res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading response from mark services handler failed %s ", err.Error())
	}
	t.Log(string(data))
}
