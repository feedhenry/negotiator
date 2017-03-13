package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/feedhenry/negotiator/deploy"
)

// TemplateHandler is the handler for the /services/template route
type TemplateHandler struct {
	templateLoader deploy.TemplateLoader
}

// NewTemplateHandler creates a template controller.
func NewTemplateHandler(tl deploy.TemplateLoader) TemplateHandler {
	return TemplateHandler{
		templateLoader: tl,
	}
}

// List all available templates in JSON format
func (s TemplateHandler) List(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-type", "application/json")
	ts, err := s.templateLoader.List()
	if err != nil {
		fmt.Println(err)
	}
	json.NewEncoder(res).Encode(ts)
}
