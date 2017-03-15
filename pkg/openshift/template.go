package openshift

import (
	"bytes"
	"fmt"

	"github.com/openshift/origin/pkg/template/api"
	"k8s.io/kubernetes/pkg/runtime"

	"path/filepath"
	"reflect"
	"text/template"

	"github.com/feedhenry/negotiator/deploy"
	"github.com/feedhenry/negotiator/pkg/openshift/templates"
	"github.com/pkg/errors"
	kapi "k8s.io/kubernetes/pkg/api"
)

// PackagedTemplates map of locally stored templates
var PackagedTemplates = map[string]string{}

func init() {
	PackagedTemplates["cloudapp"] = templates.CloudAppTemplate
	PackagedTemplates["cache"] = templates.CacheTemplate
}

func (tl *templateLoaderDecoder) Load(name string) (*template.Template, error) {

	var t = template.New("")
	t.Funcs(template.FuncMap{
		"isEnd": func(n, total int) bool {
			return n == total-1
		},
	})
	//check our own packagedTemplates first
	if localTemplate, ok := PackagedTemplates[name]; ok {
		return t.Parse(localTemplate)
	}
	//look on disk for a template
	templateFile := filepath.Join(tl.templatesDir, name+".json.tpl")
	t, err := t.ParseFiles(templateFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse template files")
	}

	return t, nil
}

type templateLoaderDecoder struct {
	templatesDir string
	decoder      runtime.Decoder
}

// NewTemplateLoaderDecoder creates a template loader that loads templates from the supplied directory
// Todo: Don't return an unexported type
func NewTemplateLoaderDecoder(templateDir string) *templateLoaderDecoder {
	return &templateLoaderDecoder{
		templatesDir: templateDir,
		decoder:      kapi.Codecs.UniversalDecoder(),
	}
}

func (tl *templateLoaderDecoder) Decode(data []byte) (*deploy.Template, error) {
	dec := tl.decoder
	obj, _, err := dec.Decode(data, nil, nil)
	if err != nil {
		fmt.Println(string(data))
		return nil, errors.Wrap(err, "failed to decode template ")
	}
	tmpl, ok := obj.(*api.Template)
	if !ok {
		kind := reflect.Indirect(reflect.ValueOf(obj)).Type().Name()
		return nil, fmt.Errorf("top level object must be of kind Template, found %s", kind)
	}

	return &deploy.Template{Template: tmpl}, tl.resolveObjects(tmpl)
}

func (tl *templateLoaderDecoder) resolveObjects(tmpl *api.Template) error {
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

func (tl *templateLoaderDecoder) List() ([]*deploy.Template, error) {
	var err error
	var t *template.Template
	var ts []*deploy.Template
	for v := range PackagedTemplates {
		t, err = tl.Load(v)
		if err != nil {
			break
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, v, &deploy.Payload{}); err != nil {
			break
		}

		decoded, err := tl.Decode(buf.Bytes())
		if err != nil {
			break
		}
		ts = append(ts, decoded)
	}
	return ts, err
}
