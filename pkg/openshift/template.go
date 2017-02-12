package openshift

import (
	"fmt"
	"k8s.io/kubernetes/pkg/runtime"
	"github.com/openshift/origin/pkg/template/api"
	"github.com/openshift/origin/pkg/template/api/v1"
	dc "github.com/openshift/origin/pkg/deploy/api"
	dc1 "github.com/openshift/origin/pkg/deploy/api/v1"
	route "github.com/openshift/origin/pkg/route/api"
	route1 "github.com/openshift/origin/pkg/route/api/v1"
	build "github.com/openshift/origin/pkg/build/api"
	build1 "github.com/openshift/origin/pkg/build/api/v1"
	image "github.com/openshift/origin/pkg/image/api"
	image1 "github.com/openshift/origin/pkg/image/api/v1"
	kapi "k8s.io/kubernetes/pkg/api"
	"reflect"
	"path/filepath"
	"text/template"
	"github.com/pkg/errors"
)

func (tl *templateLoader) Load(name string) (*template.Template, error) {
	templateFile := filepath.Join(tl.templatesDir,name + ".json.tpl")
	return template.ParseFiles(templateFile)
}

type templateLoader struct {
	templatesDir         string
	decoder             runtime.Decoder
	unstructuredDecoder runtime.Decoder
}

func NewTemplateLoader(templateDir string, )*templateLoader{

	api.AddToScheme(kapi.Scheme)
	v1.AddToScheme(kapi.Scheme)
	dc.AddToScheme(kapi.Scheme)
	dc1.AddToScheme(kapi.Scheme)
	route.AddToScheme(kapi.Scheme)
	route1.AddToScheme(kapi.Scheme)
	build.AddToScheme(kapi.Scheme)
	build1.AddToScheme(kapi.Scheme)
	image.AddToScheme(kapi.Scheme)
	image1.AddToScheme(kapi.Scheme)
	return &templateLoader{
		templatesDir:templateDir,
		decoder:kapi.Codecs.UniversalDecoder(),
		unstructuredDecoder:runtime.UnstructuredJSONScheme,
	}
}

func (tl *templateLoader) Decode(data []byte) (*api.Template, error) {
	dec := tl.decoder
	obj, _, err := dec.Decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err,"failed to decode template")
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
				return errors.Wrap(err,"failed to decode raw. Ensure to call AddToScheme")
			}
			tmpl.Objects[i] = decoded
		}
	}
	return nil
}