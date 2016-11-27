package openshift_test

import (
	"fmt"
	"testing"

	"github.com/feedhenry/negotiator/domain/openshift"
	bcv1 "github.com/openshift/origin/pkg/build/api/v1"
	ioapi "github.com/openshift/origin/pkg/image/api"
	roapi "github.com/openshift/origin/pkg/route/api"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
)

// Implements a PaasClient
type MockPassClient struct {
	Error  error
	Object interface{}
}

func (mpc MockPassClient) ListBuildConfigs(ns string) (*bcv1.BuildConfigList, error) {
	if mpc.Error != nil {
		return nil, mpc.Error
	}
	return mpc.Object.(*bcv1.BuildConfigList), nil
}
func (mpc MockPassClient) CreateServiceInNamespace(ns string, svc *api.Service) (*api.Service, error) {
	if mpc.Error != nil {
		return nil, mpc.Error
	}
	svc.Namespace = ns
	return svc, nil
}

func (mpc MockPassClient) CreateRouteInNamespace(ns string, r *roapi.Route) (*roapi.Route, error) {
	if mpc.Error != nil {
		return nil, mpc.Error
	}
	return r, nil
}

func (mpc MockPassClient) CreateImageStream(ns string, i *ioapi.ImageStream) (*ioapi.ImageStream, error) {
	if mpc.Error != nil {
		return nil, mpc.Error
	}
	return i, nil
}

func TestCreateService(t *testing.T) {
	// Create a set of test cases based on anonymous struct
	testCases := []struct {
		Name         string
		ClientError  error
		ClientObject interface{}
		Namespace    string
		ServiceName  string
		Selector     string
		Description  string
		Port         int32
		Labels       map[string]string
		Assert       func(t *testing.T, s *api.Service)
	}{
		//Actual test cases defined here
		{
			Name:         "test CreateService succeeds without error",
			ClientError:  nil,
			ClientObject: nil,
			Namespace:    "test",
			ServiceName:  "test_service",
			Selector:     "test_selector",
			Description:  "test Description",
			Port:         8080,
			Labels:       map[string]string{"testlabel": "label"},
			Assert: func(t *testing.T, s *api.Service) {
				if s == nil {
					t.Fatal("did not expect the returned service to be nil but it was ")
				}
				if s.Name != "test_service" {
					t.Errorf("expected service name to be 'test_service' but got %s ", s.Name)
				}
				if s.GetNamespace() != "test" {
					t.Errorf("expected the namespace to match 'test' but got %s", s.GetNamespace())
				}
				v, ok := s.Spec.Selector["name"]
				if !ok {
					t.Errorf("expected the service selector to have a name field %v ", v)
				}
				if v != "test_selector" {
					t.Errorf("expected the service selector to have a selector with name 'test_selector' but got %s ", v)
				}
				if nil == s.Annotations {
					t.Errorf("expected the service to have a valid set of annotations but got nil ")
				}
				v, ok = s.Annotations["rhmap/description"]
				if !ok {
					t.Errorf("expected the service annotations to have a field: 'rhmap/description' %v ", v)
				}
				if v != "test Description" {
					t.Errorf("expected the service Annotations to have description 'test Description' but got %s ", v)
				}
				if len(s.Spec.Ports) != 1 {
					t.Errorf("expected the service to have 1 port defined but got %d ", len(s.Spec.Ports))
				}
				if s.Spec.Ports[0].Port != 8080 {
					t.Errorf("epxected the service to expose port 8080 but got %d ", s.Spec.Ports[0].Port)
				}
				//awkward comparison
				if s.Spec.Ports[0].TargetPort != intstr.FromInt(int(8080)) {
					t.Errorf("epxected the service to target port 8080 but got %v ", s.Spec.Ports[0].TargetPort)
				}

			},
		},
		{
			Name:         "test CreateService succeeds without error",
			ClientError:  fmt.Errorf("unexpected service error "),
			ClientObject: nil,
			Namespace:    "test",
			ServiceName:  "test_service",
			Selector:     "test_selector",
			Description:  "test Description",
			Port:         8080,
			Labels:       map[string]string{"testlabel": "label"},
			Assert: func(t *testing.T, s *api.Service) {
				if s != nil {
					t.Fatal("did not expect a service it should have been nil ")
				}
			},
		},
	}
	// Loop through each test case and assert that we get the expected result
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			mockClient := MockPassClient{Error: tc.ClientError, Object: tc.ClientObject}
			underTest := openshift.NewService(mockClient)
			s, err := underTest.CreateService(tc.Namespace, tc.ServiceName, tc.Selector, tc.Description, tc.Port, tc.Labels)
			if tc.ClientError == nil && err != nil {
				t.Fatalf("did not expect an error but got %s ", err.Error())
			}
			if tc.ClientError != nil && err == nil {
				t.Fatal("expected an error to be returned but got none")
			}
			if tc.Assert != nil {
				tc.Assert(t, s)
			}
		})
	}
}
