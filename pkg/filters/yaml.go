package filters

import (
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type AnnotationUpdater struct {
	Key     string
	ValueFn func(string) (string, error)
}

func (f AnnotationUpdater) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
	out, err := yaml.GetAnnotation(f.Key).Filter(rn)
	if err != nil {
		return nil, err
	}

	if out.YNode().Kind != yaml.ScalarNode {
		return nil, errors.Errorf("expected annotation %s to have scalar value", f.Key)
	}

	newVal, err := f.ValueFn(yaml.GetValue(out))
	if err != nil {
		return nil, errors.WrapPrefixf(err, "error applying annotation update function")
	}

	return yaml.SetAnnotation(f.Key, newVal).Filter(rn)
}

func UpdateAnnotation(key string, valueFn func(string) (string, error)) AnnotationUpdater {
	return AnnotationUpdater{Key: key, ValueFn: valueFn}
}
