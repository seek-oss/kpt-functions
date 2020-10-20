package fns

import (
	"bytes"
	"fmt"
	"path"
	"text/template"

	"github.com/Masterminds/sprig"

	"sigs.k8s.io/kustomize/kyaml/setters2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Label used to enable template rendering
	renderTemplateEnabledLabelKey = "kpt.seek.com/render-template"

	// Label value used to enable template rendering
	renderTemplateEnabledLabelValue = "true"
)

type RenderTemplateConfig struct {
	Spec Spec `yaml:"spec,omitempty"`
}

type Spec struct {
	Kptfiles []string `yaml:"kptfiles,omitempty"`
}

type TemplateRenderer struct {
	Config          *RenderTemplateConfig
	templateContext *TemplateContext
}

type TemplateContext struct {
	Values map[string]interface{}
}

func (tr *TemplateRenderer) Filter(n *yaml.RNode) (*yaml.RNode, error) {
	if err := tr.loadTemplateContext(); err != nil {
		return nil, err
	}

	meta, err := n.GetMeta()
	if err != nil {
		return nil, err
	}

	if meta.Annotations[renderTemplateEnabledLabelKey] != renderTemplateEnabledLabelValue {
		return n, nil
	}

	return n, tr.replaceTokens(n)
}

func (tr *TemplateRenderer) replaceTokens(rn *yaml.RNode) error {
	switch rn.YNode().Kind {
	case yaml.MappingNode:
		return rn.VisitFields(func(rn *yaml.MapNode) error {
			return tr.replaceTokens(rn.Value)
		})

	case yaml.SequenceNode:
		return rn.VisitElements(func(rn *yaml.RNode) error {
			return tr.replaceTokens(rn)
		})

	case yaml.ScalarNode:
		funcMap := template.FuncMap{
			"value": func(k string) (interface{}, error) {
				if v, ok := tr.templateContext.Values[k]; ok {
					return v, nil
				}
				return nil, fmt.Errorf("template specifies missing key %s", k)
			},
		}

		tmpl, err := template.New("render").
			Funcs(sprig.TxtFuncMap()).
			Funcs(funcMap).
			Option("missingkey=error").
			Parse(rn.YNode().Value)

		if err != nil {
			return err
		}

		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, tr.templateContext); err != nil {
			return err
		}

		rn.YNode().Value = string(buf.Bytes())
	}

	return nil
}

func (tr *TemplateRenderer) loadTemplateContext() error {
	if tr.templateContext != nil {
		return nil
	}

	tr.templateContext = &TemplateContext{Values: map[string]interface{}{}}

	for _, f := range tr.Config.Spec.Kptfiles {
		// The list object will load all of the setters from the Kptfile.
		list := setters2.List{OpenAPIFileName: path.Base(f)}

		// Initialise the list object with the setter definitions from file.
		if err := list.ListSetters(f, path.Dir(f)); err != nil {
			return err
		}

		// Load each setter into the template templateContext.
		for _, s := range list.Setters {
			var value interface{}

			if len(s.ListValues) > 0 {
				value = s.ListValues
			} else if len(s.EnumValues) > 0 {
				value = s.EnumValues[s.Value]
			} else {
				value = s.Value
			}

			//tr.templateContext.Values[strcase.ToLowerCamel(s.Name)] = value
			tr.templateContext.Values[s.Name] = value
		}
	}

	return nil
}

//// Ideas:
//// Annotation that allows you to specify which fields to interpolate
//// Parse the setters from the Kptfiles
//// Allow '{{}}' syntax and specify the setter properties as Golang template variables
////
