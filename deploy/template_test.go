package deploy_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/feedhenry/negotiator/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	bc "github.com/openshift/origin/pkg/build/api"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
)

func TestDeploy(t *testing.T) {
	cases := []struct {
		TestName    string
		Template    string
		NameSpace   string
		Payload     *deploy.Payload
		Calls       map[string]int
		Asserts     map[string]func(interface{}) error
		Returns     map[string]interface{}
		ExpectError bool
	}{
		{
			ExpectError: false,
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 1,
				"CreateServiceInNamespace":      1,
			},
			Asserts: map[string]func(interface{}) error{
				"CreateDeployConfigInNamespace": func(bc interface{}) error {
					if nil == bc {
						return errors.New("did not expect nil DeploymentConfig")
					}
					dep := bc.(*dcapi.DeploymentConfig)
					if dep.Name != "cacheservice" {
						return errors.New("expected service name to be cacheservice but got " + dep.Name)
					}
					return nil
				},
				"CreateServiceInNamespace": func(s interface{}) error {
					if nil == s {
						return errors.New("did not expect nil Service")
					}
					dep := s.(*api.Service)
					if dep.Name != "cacheservice" {
						return errors.New("expected service name to be cacheservice but got " + dep.Name)
					}
					return nil
				},
			},
			TestName:  "test deploy cache",
			Template:  "cache",
			NameSpace: "test",
			Payload: &deploy.Payload{
				ServiceName:  "cacheservice",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
				},
			},
		},
		{
			TestName:    "test deploy cloudapp",
			Returns:     map[string]interface{}{"InstantiateBuild": &bc.Build{ObjectMeta: api.ObjectMeta{Name: "test"}}},
			ExpectError: false,
			Template:    "cloudapp",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				ServiceName:  "cacheservice",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
					{
						Name:  "test2",
						Value: "test2",
					},
				},
				Repo: &deploy.Repo{
					Loc: "http://git.test.com",
					Ref: "master",
				},
			},
			Calls: map[string]int{
				"CreateDeployConfigInNamespace": 1,
				"CreateServiceInNamespace":      1,
				"CreateImageStream":             1,
				"CreateBuildConfigInNamespace":  1,
				"CreateRouteInNamespace":        1,
			},
		},
		{
			TestName:    "test missing template",
			ExpectError: true,
			Template:    "idontexist",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				ServiceName: "cacheservice",
			},
		},
		{
			TestName:    "test invalid template payload",
			ExpectError: true,
			Template:    "cache",
			NameSpace:   "test",
			Payload:     &deploy.Payload{},
		},
		{
			TestName:    "test invalid cloudapp payload",
			ExpectError: true,
			Template:    "cloudapp",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				ServiceName: "test",
			},
		},
		{
			TestName: "test redeploy cloudapp",
			Returns: map[string]interface{}{
				"InstantiateBuild":            &bc.Build{ObjectMeta: api.ObjectMeta{Name: "test"}},
				"FindBuildConfigByLabel":      &bc.BuildConfig{ObjectMeta: api.ObjectMeta{Name: "test"}},
				"FindDeploymentConfigByLabel": &dcapi.DeploymentConfig{ObjectMeta: api.ObjectMeta{Name: "test"}},
			},
			ExpectError: false,
			Template:    "cloudapp",
			NameSpace:   "test",
			Payload: &deploy.Payload{
				ServiceName:  "cacheservice",
				Domain:       "rhmap",
				ProjectGUID:  "guid",
				CloudAppGUID: "guid",
				Env:          "env",
				Replicas:     1,
				EnvVars: []*deploy.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
					{
						Name:  "test2",
						Value: "test2",
					},
				},
				Repo: &deploy.Repo{
					Loc: "http://git.test.com",
					Ref: "master",
				},
			},
			Calls: map[string]int{
				"UpdateDeployConfigInNamespace": 1,
				"UpdateBuildConfigInNamespace":  1,
				"UpdateRouteInNamespace":        1,
			},
		},
	}
	tl := openshift.NewTemplateLoaderDecoder("../resources/templates/")
	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			pc := mock.NewPassClient()
			if tc.Returns != nil {
				pc.Returns = tc.Returns
			}
			pc.Asserts = tc.Asserts
			dc := deploy.New(tl, tl, logrus.StandardLogger())
			_, err := dc.Template(pc, tc.Template, tc.NameSpace, tc.Payload)
			if !tc.ExpectError && err != nil {
				fmt.Printf("%+v", err)
				t.Fatal(err)
			}
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got nil")
			}
			for f, n := range tc.Calls {
				if n != pc.CalledTimes(f) {
					t.Errorf("Expected %s to be called %d times", f, n)
				}
			}
		})
	}
}
