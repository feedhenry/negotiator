package deploy_test

import (
	"fmt"
	"github.com/feedhenry/negotiator/deploy"
	"github.com/feedhenry/negotiator/pkg/mock"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"testing"
)

func TestDeploy(t *testing.T) {
	cases := []struct {
		TestName string
		Template string
		NameSpace string
		Payload *deploy.Payload
	}{
		{
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
				EnvVars:map[string]string{"test":"val"},
			},
		},
	}
	tl := openshift.NewTemplateLoader("../../resources/templates/")
	for _, tc := range cases {
		t.Run(tc.TestName, func(t *testing.T) {
			pc := mock.PassClient{}
			dc := deploy.New(tl, tl, pc)
			if err := dc.Template(tc.Template, tc.NameSpace , tc.Payload); err != nil {
				fmt.Printf("%+v", err)
				t.Fatal(err)
			}
		})
	}
}
