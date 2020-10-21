package fns

import (
  "bytes"
  "path"
  "strings"
  "text/template"

  "github.com/Masterminds/sprig"

  "sigs.k8s.io/kustomize/kyaml/errors"
  "sigs.k8s.io/kustomize/kyaml/setters2"
  "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Annotation used to enable template rendering.
	renderTemplateEnabledAnnotationKey = "kpt.seek.com/render-template"

	// Annotation value used to enable template rendering.
	renderTemplateEnabledAnnotationValue = "true"

	// Annotation used to specify custom template delimeters.
	renderTemplateCustomDelimitersAnnotationKey = "kpt.seek.com/render-template/delimiters"
)

// TemplateRenderer provides a Kyaml filter that processes Kubernetes resources and
// renders the scalar node values as Go templates. The function config for this filter
// specifies Kptfiles whose setters are read to become the template context.
type TemplateRenderer struct {
	Config          *RenderTemplateConfig
	templateContext *TemplateContext
}

// RenderTemplateConfig is the function configuration object for TemplateRenderer.
type RenderTemplateConfig struct {
  Spec Spec `yaml:"spec,omitempty"`
}

type Spec struct {
  Kptfiles []string `yaml:"kptfiles,omitempty"`
}

// TemplateContext provides the template context that provides all of the
// values that may be accessed within templated YAML values.
type TemplateContext struct {
	Values map[string]interface{}
}

// Filter implements Kyaml's yaml.Filter.
func (tr *TemplateRenderer) Filter(n *yaml.RNode) (*yaml.RNode, error) {
	if err := tr.loadTemplateContext(); err != nil {
		return nil, err
	}

	meta, err := n.GetMeta()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if meta.Annotations[renderTemplateEnabledAnnotationKey] != renderTemplateEnabledAnnotationValue {
		return n, nil
	}

	leftDelim, rightDelim := "{{", "}}"

	if d, ok := meta.Annotations[renderTemplateCustomDelimitersAnnotationKey]; ok {
	  delims := strings.Fields(d)
	  if len(delims) != 2 {
	    return nil, errors.Errorf("%s annotation must specify a left and right delimiter separated by whitespace",
	      renderTemplateCustomDelimitersAnnotationKey)
    }

    leftDelim, rightDelim = delims[0], delims[1]
  }

	return n, tr.render(n, leftDelim, rightDelim)
}

// render recursively descends the node tree, performing template rendering on each RHS scalar value.
func (tr *TemplateRenderer) render(rn *yaml.RNode, leftDelim, rightDelim string) error {
	switch rn.YNode().Kind {
	case yaml.MappingNode:
		return rn.VisitFields(func(rn *yaml.MapNode) error {
		  // Don't attempt to render the value of the custom delimiter annotation itself,
		  // if present, as the Go template library will produce an error because the value
		  // is a set of delimiters with no command inside.
		  if rn.Key.YNode().Value == renderTemplateCustomDelimitersAnnotationKey {
        return nil
      }

      return tr.render(rn.Value, leftDelim, rightDelim)
		})

	case yaml.SequenceNode:
		return rn.VisitElements(func(rn *yaml.RNode) error {
			return tr.render(rn, leftDelim, rightDelim)
		})

	case yaml.ScalarNode:
		funcMap := template.FuncMap{
			"value": func(k string) (interface{}, error) {
				if v, ok := tr.templateContext.Values[k]; ok {
					return v, nil
				}
				return nil, errors.Errorf("template specifies missing key %s", k)
			},
		}

		tmpl, err := template.New("render").
			Funcs(sprig.TxtFuncMap()).
			Funcs(funcMap).
		  Delims(leftDelim, rightDelim).
			Option("missingkey=error").
			Parse(rn.YNode().Value)

		if err != nil {
			return errors.Wrap(err)
		}

		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, tr.templateContext); err != nil {
			return errors.Wrap(err)
		}

		rn.YNode().Value = string(buf.Bytes())
	}

	return nil
}

// loadTemplateContext reads the Kptfiles specified in the function config and
// parses all the setter key-value pairs into a cached Go template context object.
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
			return errors.Wrap(err)
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

			tr.templateContext.Values[s.Name] = value
		}
	}

	return nil
}
