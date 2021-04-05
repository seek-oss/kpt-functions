package filters

import (
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type UpdatePathFilter struct {
	// Func is the transformation function applied to the existing path annotation.
	Func func(string) (string, error)
}

// Filter implements kio.Filter.
func (f *UpdatePathFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	for _, node := range nodes {
		m, err := node.GetAnnotations()
		if err != nil {
			return nil, errors.WrapPrefixf(err, "could not get resource annotations")
		}

		path, ok := m[kioutil.PathAnnotation]
		if !ok {
			return nil, errors.Errorf("resource node is missing annotation %s: %s",
				kioutil.PathAnnotation, node.MustString())
		}

		m[kioutil.PathAnnotation], err = f.Func(path)
		if err != nil {
			return nil, err
		}

		if err := node.SetAnnotations(m); err != nil {
			return nil, err
		}
	}

	return nodes, nil
}
