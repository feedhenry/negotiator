package deploy_test

import (
	"errors"
	"testing"

	"github.com/feedhenry/negotiator/pkg/deploy"
)

type mockDeployController struct {
	Returns map[string]interface{}
}

//Template mock template method
func (m *mockDeployController) Template(client deploy.Client, namespace string, template string, payload *deploy.Payload) (*deploy.Dispatched, error) {
	if v, ok := m.Returns["Template"]; ok {
		return v.(*deploy.Dispatched), nil
	}

	return nil, errors.New("No mock return value specified for function call to Template()")
}

func test_deployDependencyServices(t *testing.T) {
	// mock_client := mock.NewPassClient()
	// mock_client.Returns = map[string]interface{}{}
	// mock_control := mockDeployController{
	// 	Returns: map[string]interface{}{
	// 		"Template": &deploy.Dispatched{},
	// 	},
	// }

}
