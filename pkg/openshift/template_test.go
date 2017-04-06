package openshift_test

import (
	"testing"

	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/openshift/origin/pkg/template/api"
	kapi "k8s.io/kubernetes/pkg/api"
)

func TestMarkServices(t *testing.T) {

	ts := []*deploy.Template{
		{
			Template: &api.Template{ObjectMeta: kapi.ObjectMeta{
				Name:   "Test",
				Labels: map[string]string{},
			},
			},
		}}

	services := []kapi.Service{
		{
			ObjectMeta: kapi.ObjectMeta{
				Name: "Test",
			},
		},
	}

	openshift.MarkServices(ts, services)

	v, ok := ts[0].Labels["deployed"]
	if !ok {
		t.Fatal("expected deployed key")
	}
	if v != "true" {
		t.Fatal("expected value true")
	}
}
