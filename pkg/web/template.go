package web

import (
	"encoding/json"
	"net/http"

	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/prometheus/common/log"
)

// TemplateHandler is the handler for the /services/template route
type TemplateHandler struct {
	templateLoader deploy.TemplateLoader
	clientFactory  DeployClientFactory
}

// NewTemplateHandler creates a template controller.
func NewTemplateHandler(tl deploy.TemplateLoader, clientFactory DeployClientFactory) TemplateHandler {
	return TemplateHandler{
		templateLoader: tl,
		clientFactory:  clientFactory,
	}
}

// MarkServices  all service templates that are deployed in an environment (label 'deployed' to true)
func (s TemplateHandler) MarkServices(res http.ResponseWriter, req *http.Request) {
	// get a list of all templates
	res.Header().Add("Content-type", "application/json")
	ts, err := s.templateLoader.ListServices()
	if err != nil {
		log.Error(err)
	}

	ns := req.URL.Query().Get("env")
	if ns != "" {
		// get a list of deployed templates
		host := req.Header.Get("X-RHMAP-HOST")
		token := req.Header.Get("X-RHMAP-TOKEN")

		if host == "" || token == "" {
			// we just log as error and return the templates 'unmarked'
			log.Error("headers for host/token are missing")
			json.NewEncoder(res).Encode(ts)
			return
		}

		client, err := s.clientFactory.DefaultDeployClient(host, token)
		if err != nil {
			http.Error(res, err.Error(), http.StatusUnauthorized)
			return
		}

		services, err := client.FindServiceByLabel(ns, map[string]string{"rhmap/type": "environmentService"})

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		openshift.MarkServices(ts, services)

	}

	if err := json.NewEncoder(res).Encode(ts); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
