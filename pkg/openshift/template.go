package openshift

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/openshift/origin/pkg/template/api"
	"github.com/prometheus/common/log"
	"k8s.io/kubernetes/pkg/runtime"

	"path/filepath"
	"reflect"
	"text/template"

	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/openshift/templates"
	"github.com/pkg/errors"
	kapi "k8s.io/kubernetes/pkg/api"
)

// PackagedTemplates map of locally stored templates
var PackagedTemplates = map[string]string{}
var ServiceTemplates = map[string]string{}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	PackagedTemplates["cloudapp"] = templates.CloudAppTemplate
	PackagedTemplates["cache-redis"] = templates.CacheRedisTemplate
	PackagedTemplates["data-mongo"] = templates.DataMongoTemplate
	PackagedTemplates["data-mongo-replica"] = templates.DataMongoReplicaTemplate
	PackagedTemplates["data-mongo-job"] = templates.DataMongoConfigJob
	PackagedTemplates["data-mysql"] = templates.DataMySQLTemplate
	PackagedTemplates["data-mysql-job"] = templates.DataMysqlConfigJob
	PackagedTemplates["push-ups"] = templates.PushUPSTemplate

	ServiceTemplates["cloudapp"] = templates.CloudAppTemplate
	ServiceTemplates["cache-redis"] = templates.CacheRedisTemplate
	ServiceTemplates["data-mongo"] = templates.DataMongoTemplate
	ServiceTemplates["data-mongo-replica"] = templates.DataMongoReplicaTemplate
	ServiceTemplates["data-mysql"] = templates.DataMySQLTemplate
	ServiceTemplates["push-ups"] = templates.PushUPSTemplate
}

func (tl *templateLoaderDecoder) Load(name string) (*template.Template, error) {

	var t = template.New("")
	var genPass = func() string {
		b := make([]byte, 16)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}
		return string(b)
	}
	var generatedPass = genPass() // a single generated password per template load
	t.Funcs(template.FuncMap{
		"isEnd": func(n, total int) bool {
			return n == total-1
		},
		"genPass": genPass,
		"generatedPass": func() string {
			return generatedPass
		},
		"isset": func(vals map[string]interface{}, key string) bool {
			if nil == vals {
				return false
			}
			_, ok := vals[key]
			return ok
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
		return nil, errors.Wrap(err, "failed to decode template ")
	}
	tmpl, ok := obj.(*api.Template)
	if !ok {
		kind := reflect.Indirect(reflect.ValueOf(obj)).Type().Name()
		return nil, fmt.Errorf("top level object must be of kind Template, found %s", kind)
	}
	errs := runtime.DecodeList(tmpl.Objects, kapi.Codecs.UniversalDecoder())
	if len(errs) > 0 {
		return nil, errs[0]
	}

	return &deploy.Template{Template: tmpl}, nil
}

func (tl *templateLoaderDecoder) ListServices() ([]*deploy.Template, error) {
	var err error
	var t *template.Template
	var ts []*deploy.Template
	for v := range ServiceTemplates {
		t, err = tl.Load(v)
		if err != nil {
			log.Error(err.Error())
			break
		}
		var buf bytes.Buffer
		if err = t.ExecuteTemplate(&buf, v, &deploy.Payload{}); err != nil {
			break
		}
		var decoded *deploy.Template
		decoded, err = tl.Decode(buf.Bytes())
		if err != nil {
			break
		}
		ts = append(ts, decoded)
	}
	return ts, err
}

func (tl *templateLoaderDecoder) FindInTemplate(t *api.Template, resource string) (interface{}, error) {
	for _, obj := range t.Objects {
		if resource == reflect.Indirect(reflect.ValueOf(obj)).Type().Name() {
			return obj, nil
		}

	}
	return nil, errors.New("resource not found in template " + resource)
}

// MarkServices mark services that are deployed to true
func MarkServices(ts []*deploy.Template, services []kapi.Service) {
	for _, template := range ts {
		for _, service := range services {
			if service.Name == template.Name {
				//if _, ok := template.Template.Labels["deployed"]; ok {
				template.Template.Labels["deployed"] = "true"
				log.Info("Template %s deployed set to true ", template.Name)
				//}
			}
		}
	}
}
