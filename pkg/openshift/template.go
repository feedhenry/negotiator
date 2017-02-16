package openshift

import (
	"fmt"

	"github.com/openshift/origin/pkg/template/api"
	"k8s.io/kubernetes/pkg/runtime"

	"path/filepath"
	"reflect"
	"text/template"

	"github.com/pkg/errors"
	kapi "k8s.io/kubernetes/pkg/api"
)

func (tl *templateLoader) Load(name string) (*template.Template, error) {
	templateFile := filepath.Join(tl.templatesDir, name+".json.tpl")
	t := template.New("")
	t.Funcs(template.FuncMap{
		"isEnd": func(n, total int) bool {
			return n == total-1
		},
	})
	t, err := t.ParseFiles(templateFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse template files")
	}

	return t, nil
}

type templateLoader struct {
	templatesDir        string
	decoder             runtime.Decoder
	unstructuredDecoder runtime.Decoder
}

func NewTemplateLoader(templateDir string) *templateLoader {
	return &templateLoader{
		templatesDir:        templateDir,
		decoder:             kapi.Codecs.UniversalDecoder(),
		unstructuredDecoder: runtime.UnstructuredJSONScheme,
	}
}

func (tl *templateLoader) Decode(data []byte) (*api.Template, error) {
	dec := tl.decoder
	obj, _, err := dec.Decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode template")
	}
	tmpl, ok := obj.(*api.Template)
	if !ok {
		kind := reflect.Indirect(reflect.ValueOf(obj)).Type().Name()
		return nil, errors.New(fmt.Sprintf("top level object must be of kind Template, found %s", kind))
	}

	return tmpl, tl.resolveObjects(tmpl)
}

func (tl *templateLoader) resolveObjects(tmpl *api.Template) error {
	dec := tl.decoder

	for i, obj := range tmpl.Objects {
		if unknown, ok := obj.(*runtime.Unknown); ok {
			decoded, _, err := dec.Decode(unknown.Raw, nil, nil)
			if err != nil {
				return errors.Wrap(err, "failed to decode raw. Ensure to call AddToScheme")
			}
			tmpl.Objects[i] = decoded
		}
	}
	return nil
}
