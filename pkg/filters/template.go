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
	templateEnabledKey    = "$kpt-template"
	templateEnabledValue  = "true"
	leftDelimiterKey      = "$kpt-template-left-delimiter"
	rightDelimiterKey     = "$kpt-template-right-delimiter"
	defaultLeftDelimiter  = "{{"
	defaultRightDelimiter = "}}"
)

type delimiter int

const (
	left  delimiter = iota
	right           = iota
)

// TemplateParameters holds the parameters that are declared in the json comment that indicates that templating
// of a field is required. This allows us to propagate things like the desired delimiters and whether template
// expansion is enabled, as part of walking the yaml tree.
type TemplateParameters struct {
	templatingEnabled bool
	leftDelimiter     string
	rightDelimiter    string
}

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

type loadFn func(leftDelimiter, rightDelimiter string) (template *gotemplate.Template, templateContext *TemplateContext, error error)

// Filter implements Kyaml's yaml.Filter.
func (f *TemplateFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	kptfileNodes, err := KptfileFilter().Filter(nodes)
	if err != nil {
		return nil, err
	}

	if len(kptfileNodes) != 1 {
		return nil, errors.Errorf("expected a single Kptfile in package but got %d", len(kptfileNodes))
	}

	loadFn := func(leftDelimiter, rightDelimiter string) (template *gotemplate.Template, templateContext *TemplateContext, error error) {
		return f.load(kptfileNodes[0], leftDelimiter, rightDelimiter)
	}

	notKptfileNodes, err := NotKptfileFilter().Filter(nodes)
	if err != nil {
		return nil, err
	}

	for _, node := range notKptfileNodes {
		if err := f.process(node, loadFn); err != nil {
			return nil, err
		}
	}

	return nodes, nil
}

// process processes the templated fields in the specified resource node.
func (f *TemplateFilter) process(resource *yaml.RNode, loadFn loadFn) error {
	exec := func(v, leftDelimiter, rightDelimiter string) (string, error) {
		temp, templateContext, err := loadFn(leftDelimiter, rightDelimiter)
		if err != nil {
			return "", errors.Wrap(err)
		}
		tmpl, err := temp.Clone()
		if err != nil {
			return "", errors.Wrap(err)
		}

		tmpl.Delims(leftDelimiter, rightDelimiter)
		if _, err := tmpl.Parse(v); err != nil {
			return "", errors.Wrap(err)
		}

		buf := bytes.Buffer{}
		if err := tmpl.Execute(&buf, templateContext); err != nil {
			return "", errors.Wrap(err)
		}

		return buf.String(), nil
	}

	return f.render(resource, exec, TemplateParameters{false, defaultLeftDelimiter, defaultRightDelimiter})
}

type executeTemplate func(string, string, string) (string, error)

// render recursively descends the node tree, performing template rendering on each RHS scalar value.
func (f *TemplateFilter) render(rn *yaml.RNode, exec executeTemplate, templateParameters TemplateParameters) error {
	templateParameters.templatingEnabled = templateParameters.templatingEnabled || f.hasEnabledTemplating(rn)
	if templateParameters.leftDelimiter == defaultLeftDelimiter {
		templateParameters.leftDelimiter = f.getDelimiter(rn, left)
	}
	if templateParameters.rightDelimiter == defaultRightDelimiter {
		templateParameters.rightDelimiter = f.getDelimiter(rn, right)
	}
	switch rn.YNode().Kind {
	case yaml.MappingNode:
		return rn.VisitFields(func(rn *yaml.MapNode) error {
			templateParameters.templatingEnabled = f.hasEnabledTemplating(rn.Key)
			if templateParameters.leftDelimiter == defaultLeftDelimiter {
				templateParameters.leftDelimiter = f.getDelimiter(rn.Key, left)
			}
			if templateParameters.rightDelimiter == defaultRightDelimiter {
				templateParameters.rightDelimiter = f.getDelimiter(rn.Key, right)
			}
			return f.render(rn.Value, exec, templateParameters)
		})

	case yaml.SequenceNode:
		return rn.VisitElements(func(rn *yaml.RNode) error {
			return f.render(rn, exec, templateParameters)
		})

	case yaml.ScalarNode:
		if templateParameters.templatingEnabled {
			res, err := exec(rn.YNode().Value, templateParameters.leftDelimiter, templateParameters.rightDelimiter)
			if err != nil {
				return err
			}

			rn.YNode().Value = res
		}
	}

	return nil
}

func (f *TemplateFilter) load(kptfile *yaml.RNode, leftDelimiter, rightDelimiter string) (*gotemplate.Template, *TemplateContext, error) {
	templateContext, err := f.loadTemplateContext(kptfile)
	if err != nil {
		return nil, nil, err
	}

	template := gotemplate.New("render").
		Funcs(sprig.TxtFuncMap()).
		Option("missingkey=error").
		Delims(leftDelimiter, rightDelimiter)

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

		return buf.String(), nil
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

// getLeftDelimiter returns the left delimiter specified in the YAML head annotation
func (f *TemplateFilter) getDelimiter(n *yaml.RNode, delimiter delimiter) string {
	comments := []string{n.YNode().LineComment, n.YNode().HeadComment}
	for _, c := range comments {
		if c == "" {
			continue
		}

		m := map[string]string{}
		err := json.Unmarshal([]byte(strings.TrimLeft(c, "#")), &m)
		if err != nil {
			return defaultDelimiter(delimiter)
		}

		if delimiter == left {
			if delim, ok := m[leftDelimiterKey]; ok {
				return delim
			}
		} else {
			if delim, ok := m[rightDelimiterKey]; ok {
				return delim
			}
		}
	}

	return defaultDelimiter(delimiter)
}

func defaultDelimiter(delimiter delimiter) string {
	if delimiter == left {
		return defaultLeftDelimiter
	} else {
		return defaultRightDelimiter
	}
}
