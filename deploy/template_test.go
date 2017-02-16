package deploy_test

import (
	"fmt"
	"github.com/feedhenry/negotiator/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"testing"
	"github.com/pkg/errors"
	dcapi "github.com/openshift/origin/pkg/deploy/api"
	"k8s.io/kubernetes/pkg/api"
)

func TestDeploy(t *testing.T) {
	cases := []struct {
		TestName string
		Template string
		NameSpace string
		Payload *deploy.Payload
		Calls map[string]int
		Asserts map[string]func(interface{})error
		ExpectError bool
	}{
		{
			ExpectError:false,
			Calls: map[string]int{
				"CreateDeployConfigInNamespace":1,
				"CreateServiceInNamespace":1,
			},
			Asserts: map[string]func(interface{})error{
				"CreateDeployConfigInNamespace": func(bc interface{})error{
					if nil == bc{
						return errors.New("did not expect nil DeploymentConfig")
					}
					dep := bc.(*dcapi.DeploymentConfig)
					if dep.Name != "cacheservice"{
						return errors.New("expected service name to be cacheservice but got " + dep.Name)
					}
					return nil
				},
				"CreateServiceInNamespace": func (s interface{})error{
					if nil == s{
						return errors.New("did not expect nil Service")
					}
					dep := s.(*api.Service)
					if dep.Name != "cacheservice"{
						return errors.New("expected service name to be cacheservice but got " + dep.Name)
					}
					return nil
				},
			},
			TestName: "test deploy cache",
			Template:"cache",
			NameSpace:"test",
			Payload:&deploy.Payload{
				ServiceName:"cacheservice",
				Domain:"rhmap",
				ProjectGuid:"guid",
				CloudAppGuid:"guid",
				Env:"env",
				Replicas:1,
				EnvVars:[]*deploy.EnvVar{
					{
						Name:"test",
						Value:"test",
					},

				},
			},
		},
		{
			TestName:"test deploy cloudapp",
			ExpectError:false,
			Template:"cloudapp",
			NameSpace:"test",
			Payload:&deploy.Payload{
				ServiceName:"cacheservice",
				Domain:"rhmap",
				ProjectGuid:"guid",
				CloudAppGuid:"guid",
				Env:"env",
				Replicas:1,
				EnvVars:[]*deploy.EnvVar{
					{
						Name:"test",
						Value:"test",
					},
					{
						Name:"test2",
						Value:"test2",
					},

				},
				Repo:&deploy.Repo{
					Loc:"http://git.test.com",
					Ref:"master",
				},
			},
			Calls: map[string]int{
				"CreateDeployConfigInNamespace":1,
				"CreateServiceInNamespace":1,
				"CreateImageStream":1,
				"CreateBuildConfigInNamespace":1,
				"CreateRouteInNamespace":1,
			},
		},
		{
			TestName:"test missing template",
			ExpectError:true,
			Template:"idontexist",
			NameSpace:"test",
		},
	}
	tl := openshift.NewTemplateLoader("../resources/templates/")
	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			pc := mock.NewPassClient()
			pc.Asserts = tc.Asserts
			dc := deploy.New(tl, tl, pc)
			err := dc.Template(tc.Template, tc.NameSpace , tc.Payload)
			if ! tc.ExpectError && err != nil {
				fmt.Printf("%+v", err)
				t.Fatal(err)
			}
			if tc.ExpectError && err == nil{
				t.Fatal("expected an error but got nil")
			}
			for f, n := range tc.Calls{
				if n != pc.CalledTimes(f){
					t.Errorf("Excpected %s to be called %d times", f,n)
				}
			}
		})
	}
}
