package filters

import (
	"encoding/json"

	"sigs.k8s.io/kustomize/kyaml/kio"

	"sigs.k8s.io/kustomize/kyaml/errors"

	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/setters2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetPackageFilter provides a kio.Filter implementation that executes a setter on Kpt packages.
// On each invocation of the Filter function, SetPackageFilter expects to be given a single Kpt
// package, where exactly one of the resource nodes pertains to the Kptfile.
type SetPackageFilter struct {
	// Name is the name of the setter.
	Name string
	// Value is the single value of the setter.
	Value string
	// ListValue is the list value of the setter.
	ListValues []string
	// SetBy specifies who executed the setter.
	SetBy string
}

// Filter implements kio.Filter.
func (f *SetPackageFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	kptfileNodes, err := KptfileFilter().Filter(nodes)
	if err != nil {
		return nil, err
	}

	if len(kptfileNodes) != 1 {
		return nil, errors.Errorf("expected a single Kptfile in package but got %d", len(kptfileNodes))
	}

	notKptfileNodes, err := NotKptfileFilter().Filter(nodes)
	if err != nil {
		return nil, err
	}

	kptfileNodes, err = f.kptfileSetterFilter().Filter(kptfileNodes)
	if err != nil {
		return nil, err
	}

	notKptfileNodes, err = f.resourceSetterFilter(kptfileNodes[0]).Filter(notKptfileNodes)
	if err != nil {
		return nil, err
	}

	return append(kptfileNodes, notKptfileNodes...), nil
}

// kptfileSetterFilter returns a kio.Filter that invokes a setter on Kptfile resource nodes.
func (f *SetPackageFilter) kptfileSetterFilter() kio.Filter {
	return kio.FilterAll(yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		if !hasKptfileSetter(node, f.Name) {
			return node, nil
		}

		if len(f.ListValues) > 0 {
			f.Value = f.ListValues[0]
			f.ListValues = f.ListValues[1:]
		}

		return setters2.SetOpenAPI{
			Name:       f.Name,
			Value:      f.Value,
			ListValues: f.ListValues,
			SetBy:      f.SetBy,
			IsSet:      true,
		}.Filter(node)
	}))
}

// resourceSetterFilter returns a kio.Filter that invokes a setter on regular (i.e., non-Kptfile) resource nodes.
func (f *SetPackageFilter) resourceSetterFilter(kptfile *yaml.RNode) kio.Filter {
	schema, err := openAPISchema(kptfile)
	if err != nil {
		panic(err)
	}

	return kio.FilterAll(&setters2.Set{
		Name:          f.Name,
		SettersSchema: schema,
	})
}

// openAPISchema returns the spec.Schema for the specified Kptfile resource node.
func openAPISchema(kptfile *yaml.RNode) (*spec.Schema, error) {
	m := kptfile.Field(openapi.SupplementaryOpenAPIFieldName)
	if m.IsNilOrEmpty() {
		// doesn't contain openAPI definitions
		return nil, nil
	}
	kptfile = m.Value

	oa, err := kptfile.String()
	if err != nil {
		return nil, err
	}

	var o interface{}
	err = yaml.Unmarshal([]byte(oa), &o)
	if err != nil {
		return nil, err
	}
	j, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	var sc spec.Schema
	err = sc.UnmarshalJSON(j)
	if err != nil {
		return nil, err
	}

	return &sc, nil
}
