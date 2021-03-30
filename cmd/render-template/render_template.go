package main

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
	Config *RenderTemplateConfig

	template        *template.Template
	templateContext *TemplateContext
}

// RenderTemplateConfig is the function configuration object for TemplateRenderer.
type RenderTemplateConfig struct {
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`

	Spec struct {
		Kptfiles []string `yaml:"kptfiles,omitempty"`
	} `yaml:"spec"`
}

// TemplateContext provides the template context that provides all of the
// values that may be accessed within templated YAML values.
type TemplateContext struct {
	Values map[string]interface{}
}

// Filter implements Kyaml's yaml.Filter.
func (tr *TemplateRenderer) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	for _, node := range nodes {
		if err := tr.process(node); err != nil {
			return nil, err
		}
	}

	return nodes, nil
}

// Filter implements Kyaml's yaml.Filter.
func (tr *TemplateRenderer) process(n *yaml.RNode) error {
	if err := tr.init(); err != nil {
		return err
	}

	meta, err := n.GetMeta()
	if err != nil {
		return errors.Wrap(err)
	}

	if meta.Annotations[renderTemplateEnabledAnnotationKey] != renderTemplateEnabledAnnotationValue {
		return nil
	}

	leftDelim, rightDelim := "{{", "}}"

	if d, ok := meta.Annotations[renderTemplateCustomDelimitersAnnotationKey]; ok {
		delims := strings.Fields(d)
		if len(delims) != 2 {
			return errors.Errorf("%s annotation must specify a left and right delimiter separated by whitespace",
				renderTemplateCustomDelimitersAnnotationKey)
		}

		leftDelim, rightDelim = delims[0], delims[1]
	}

	exec := func(v string) (string, error) {
		tmpl, err := tr.template.Clone()
		if err != nil {
			return "", errors.Wrap(err)
		}

		tmpl.Delims(leftDelim, rightDelim)

		if _, err := tmpl.Parse(v); err != nil {
			return "", errors.Wrap(err)
		}

		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, tr.templateContext); err != nil {
			return "", errors.Wrap(err)
		}

		return string(buf.Bytes()), nil
	}

	return tr.render(n, exec)
}

type executeTemplate func(string) (string, error)

// render recursively descends the node tree, performing template rendering on each RHS scalar value.
func (tr *TemplateRenderer) render(rn *yaml.RNode, exec executeTemplate) error {
	switch rn.YNode().Kind {
	case yaml.MappingNode:
		return rn.VisitFields(func(rn *yaml.MapNode) error {
			// Don't attempt to render the value of the custom delimiter annotation itself,
			// if present, as the Go template library will produce an error because the value
			// is a set of delimiters with no command inside.
			if rn.Key.YNode().Value == renderTemplateCustomDelimitersAnnotationKey {
				return nil
			}

			return tr.render(rn.Value, exec)
		})

	case yaml.SequenceNode:
		return rn.VisitElements(func(rn *yaml.RNode) error {
			return tr.render(rn, exec)
		})

	case yaml.ScalarNode:
		res, err := exec(rn.YNode().Value)
		if err != nil {
			return err
		}

		rn.YNode().Value = res
	}

	return nil
}

func (tr *TemplateRenderer) init() error {
	if tr.template != nil {
		return nil
	}

	if err := tr.loadTemplate(); err != nil {
		return err
	}

	return tr.loadTemplateContext()
}

func (tr *TemplateRenderer) loadTemplate() error {
	tr.template = template.New("render").
		Funcs(sprig.TxtFuncMap()).
		Option("missingkey=error")

	valueFn := func(k string) (interface{}, error) {
		if v, ok := tr.templateContext.Values[k]; ok {
			return v, nil
		}
		return nil, errors.Errorf("template specifies missing key %s", k)
	}

	var renderFn func(tk string, args ...string) (string, error)
	renderFn = func(tk string, args ...string) (string, error) {
		tv, err := valueFn(tk)
		if err != nil {
			return "", err
		}

		text, ok := tv.(string)
		if !ok {
			return "", errors.Errorf("referenced template '%s' is not a string", tk)
		}

		newTmpl, err := tr.template.Clone()
		if err != nil {
			return "", errors.Wrap(err)
		}

		argsFn := func(i int) string {
			return args[i]
		}

		newTmpl.Funcs(template.FuncMap{"value": valueFn, "render": renderFn, "args": argsFn})

		if _, err := newTmpl.Parse(text); err != nil {
			return "", errors.Wrap(err)
		}

		buf := bytes.Buffer{}
		if err := newTmpl.Execute(&buf, tr.templateContext); err != nil {
			return "", errors.Wrap(err)
		}

		return string(buf.Bytes()), nil
	}

	tr.template.Funcs(template.FuncMap{"value": valueFn, "render": renderFn})

	return nil
}

// loadTemplateContext reads the Kptfiles specified in the function config and
// parses all the setter key-value pairs into a cached Go template context object.
func (tr *TemplateRenderer) loadTemplateContext() error {
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
