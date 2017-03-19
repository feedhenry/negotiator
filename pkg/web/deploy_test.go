package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/kubernetes/pkg/api"

	"strings"

	"time"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/feedhenry/negotiator/pkg/web"
	bc "github.com/openshift/origin/pkg/build/api"
)

func setUpDeployHandler() http.Handler {
	//map[string]interface{}{"InstantiateBuild": &bc.Build{ObjectMeta: api.ObjectMeta{Name: "test"}}},
	pc := mock.NewPassClient()
	pc.Returns = map[string]interface{}{"InstantiateBuild": &bc.Build{ObjectMeta: api.ObjectMeta{Name: "test"}}}
	clientFactory := mock.ClientFactory{PassClient: pc}
	router := web.BuildRouter()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logger := logrus.StandardLogger()
	templates := openshift.NewTemplateLoaderDecoder("")
	serviceConfigFactory := &deploy.ConfigurationFactory{StatusPublisher: deploy.LogStatusPublisher{Logger: logger}}
	serviceConfigController := deploy.NewEnvironmentServiceConfigController(serviceConfigFactory, logger, nil, templates)
	deployController := deploy.New(templates, templates, logger, serviceConfigController)
	web.DeployRoute(router, logger, deployController, clientFactory)
	return web.BuildHTTPHandler(router)
}

func TestDeploys(t *testing.T) {
	cases := []struct {
		Name       string
		Body       string
		StatusCode int
		Template   string
		NameSpace  string
	}{
		{
			Name:       "test deploy cloudapp",
			Body:       `{"repo": {"loc": "https://github.com/feedhenry/testing-cloud-app.git","ref": "master"}, "target":{"host":"https://notthere.com:8443","token":"token"}, "serviceName": "cloudapp4","replicas": 1,  "projectGuid":"test","envVars":[{"name":"test","value":"test"}]}`,
			StatusCode: 200,
			Template:   "cloudapp",
			NameSpace:  "test",
		},
		{
			Name:       "test deploy no template",
			Body:       `{"repo": {"loc": "https://github.com/feedhenry/testing-cloud-app.git","ref": "master"}, "target":{"host":"https://notthere.com:8443","token":"token"}, "serviceName": "cloudapp4","replicas": 1,  "projectGuid":"test","envVars":[{"name":"test","value":"test"}]}`,
			StatusCode: 404,
			Template:   "notthere",
			NameSpace:  "test",
		},
		{
			Name:       "test deploy no cache",
			Body:       `{"target":{"host":"https://notthere.com:8443","token":"token"}, "serviceName": "cloudapp4","replicas": 1,  "projectGuid":"test","envVars":[{"name":"test","value":"test"}]}`,
			StatusCode: 200,
			Template:   "cache",
			NameSpace:  "test",
		},
		{
			Name:       "test cloud app requires repo",
			Body:       `{"target":{"host":"https://notthere.com:8443","token":"token"}, "serviceName": "cloudapp4","replicas": 1,  "projectGuid":"test","envVars":[{"name":"test","value":"test"}]}`,
			StatusCode: 400,
			Template:   "cloudapp",
			NameSpace:  "test",
		},
	}

	server := httptest.NewServer(setUpDeployHandler())
	defer server.Close()
	http.DefaultClient.Timeout = time.Second * 10
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			url := server.URL + "/service/deploy/" + tc.Template + "/" + tc.NameSpace
			r, err := http.NewRequest("POST", url, strings.NewReader(tc.Body))
			if err != nil {
				t.Fatalf("failed to create request %s ", err.Error())
			}
			res, err := http.DefaultClient.Do(r)
			if err != nil {
				t.Fatalf("failed to make request againt %s error %s", url, err.Error())
			}
			defer res.Body.Close()
			if res.StatusCode != tc.StatusCode {
				t.Fatalf("expected status code %d but got %d ", tc.StatusCode, res.StatusCode)
			}
		})
	}

}
