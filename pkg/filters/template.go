package filters

import (
	"bytes"
	"encoding/json"
	"strings"
	gotemplate "text/template"

	"sigs.k8s.io/kustomize/kyaml/fieldmeta"

	"sigs.k8s.io/kustomize/kyaml/openapi"

	"github.com/Masterminds/sprig"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/setters2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	templateEnabledKey   = "$kpt-template"
	templateEnabledValue = "true"
)

// TemplateFilter provides a Kyaml filter that processes Kubernetes resources and  renders the scalar node values
// as Go templates. The function config for this filter specifies Kptfiles whose setters are read to become the
// template context. On each invocation of the Filter function, SetPackageFilter expects to be given a single Kpt
// package, where exactly one of the resource nodes pertains to the Kptfile.
type TemplateFilter struct{}

// TemplateContext provides the template context that provides all of the
// values that may be accessed within templated YAML values.
type TemplateContext struct {
	Values map[string]interface{}
}

// Filter implements Kyaml's yaml.Filter.
func (f *TemplateFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	kptfileNodes, err := KptfileFilter().Filter(nodes)
	if err != nil {
		return nil, err
	}

	if len(kptfileNodes) != 1 {
		return nil, errors.Errorf("expected a single Kptfile in package but got %d", len(kptfileNodes))
	}

	template, templateContext, err := f.load(kptfileNodes[0])
	if err != nil {
		return nil, err
	}

	notKptfileNodes, err := NotKptfileFilter().Filter(nodes)
	if err != nil {
		return nil, err
	}

	for _, node := range notKptfileNodes {
		if err := f.process(node, template, templateContext); err != nil {
			return nil, err
		}
	}

	return nodes, nil
}

// process processes the templated fields in the specified resource node.
func (f *TemplateFilter) process(resource *yaml.RNode, template *gotemplate.Template, templateContext *TemplateContext) error {
	exec := func(v string) (string, error) {
		tmpl, err := template.Clone()
		if err != nil {
			return "", errors.Wrap(err)
		}

		if _, err := tmpl.Parse(v); err != nil {
			return "", errors.Wrap(err)
		}

		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, templateContext); err != nil {
			return "", errors.Wrap(err)
		}

		return string(buf.Bytes()), nil
	}

	return f.render(resource, exec, false)
}

type executeTemplate func(string) (string, error)

// render recursively descends the node tree, performing template rendering on each RHS scalar value.
func (f *TemplateFilter) render(rn *yaml.RNode, exec executeTemplate, templatingEnabled bool) error {
	templatingEnabled = templatingEnabled || f.hasEnabledTemplating(rn)
	switch rn.YNode().Kind {
	case yaml.MappingNode:
		return rn.VisitFields(func(rn *yaml.MapNode) error {
			return f.render(rn.Value, exec, templatingEnabled || f.hasEnabledTemplating(rn.Key))
		})

	case yaml.SequenceNode:
		return rn.VisitElements(func(rn *yaml.RNode) error {
			return f.render(rn, exec, templatingEnabled)
		})

	case yaml.ScalarNode:
		if templatingEnabled {
			res, err := exec(rn.YNode().Value)
			if err != nil {
				return err
			}

			rn.YNode().Value = res
		}
	}

	return nil
}

func (f *TemplateFilter) load(kptfile *yaml.RNode) (*gotemplate.Template, *TemplateContext, error) {
	templateContext, err := f.loadTemplateContext(kptfile)
	if err != nil {
		return nil, nil, err
	}

	template := gotemplate.New("render").
		Funcs(sprig.TxtFuncMap()).
		Option("missingkey=error")

	valueFn := func(k string) (interface{}, error) {
		if v, ok := templateContext.Values[k]; ok {
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

		newTmpl, err := template.Clone()
		if err != nil {
			return "", errors.Wrap(err)
		}

		argsFn := func(i int) string {
			return args[i]
		}

		nargsFn := func() int {
			return len(args)
		}

		newTmpl.Funcs(gotemplate.FuncMap{"value": valueFn, "render": renderFn, "args": argsFn, "nargs": nargsFn})

		if _, err := newTmpl.Parse(text); err != nil {
			return "", errors.Wrap(err)
		}

		buf := bytes.Buffer{}
		if err := newTmpl.Execute(&buf, templateContext); err != nil {
			return "", errors.Wrap(err)
		}

		return string(buf.Bytes()), nil
	}

	template.Funcs(gotemplate.FuncMap{"value": valueFn, "render": renderFn})

	return template, templateContext, nil
}

// loadTemplateContext reads the Kptfiles specified in the function config and
// parses all the setter key-value pairs into a cached Go template context object.
func (f *TemplateFilter) loadTemplateContext(kptfile *yaml.RNode) (*TemplateContext, error) {
	templateContext := &TemplateContext{Values: map[string]interface{}{}}

	setters, err := f.listSetters(kptfile)
	if err != nil {
		return nil, err
	}

	// Load each setter into the template templateContext.
	for _, s := range setters {
		var value interface{}

		if len(s.ListValues) > 0 {
			value = s.ListValues
		} else if len(s.EnumValues) > 0 {
			value = s.EnumValues[s.Value]
		} else {
			value = s.Value
		}

		templateContext.Values[s.Name] = value
	}

	return templateContext, nil
}

func (f *TemplateFilter) listSetters(kptfile *yaml.RNode) ([]setters2.SetterDefinition, error) {
	defs, err := kptfile.Pipe(yaml.Lookup(openapi.SupplementaryOpenAPIFieldName, openapi.Definitions))
	if err != nil {
		return nil, err
	}

	var setters []setters2.SetterDefinition

	if err := defs.VisitFields(func(node *yaml.MapNode) error {
		setter := setters2.SetterDefinition{}
		key := node.Key.YNode().Value

		if !strings.HasPrefix(key, fieldmeta.SetterDefinitionPrefix) {
			// Not a setter as it doesn't have the right prefix.
			return nil
		}

		setterNode, err := node.Value.Pipe(yaml.Lookup(setters2.K8sCliExtensionKey, "setter"))
		if err != nil {
			return err
		}
		if yaml.IsMissingOrNull(setterNode) {
			// Has the setter prefix, but missing the setter extension.
			return errors.Errorf("missing x-k8s-cli.setter for %s", key)
		}

		b, err := setterNode.String()
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal([]byte(b), &setter); err != nil {
			return err
		}

		setters = append(setters, setter)
		return nil
	}); err != nil {
		return nil, err
	}

	return setters, nil
}

// hasEnabledTemplating returns whether the specified YAML node has enabled templating via a
// {"$kpt-template":"true"} annotation.
func (f *TemplateFilter) hasEnabledTemplating(n *yaml.RNode) bool {
	comments := []string{n.YNode().LineComment, n.YNode().HeadComment}
	for _, c := range comments {
		if c == "" {
			continue
		}

		m := map[string]string{}
		err := json.Unmarshal([]byte(strings.TrimLeft(c, "#")), &m)
		if err != nil {
			return false
		}

		if m[templateEnabledKey] == templateEnabledValue {
			return true
		}
	}

	return false
}
