package filters

import (
	"github.com/go-errors/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"testing"
)

func newProcessor(renderer kio.Filter) framework.ResourceListProcessor {
	return framework.SimpleProcessor{
		Filter: renderer,
	}
}

func normaliseYAML(doc string) string {
	return yaml.MustParse(doc).MustString()
}

func fatalError(t *testing.T, err error) {
	t.Helper()

	if e, ok := err.(*errors.Error); ok {
		trace := e.ErrorStack()
		t.Fatal(err, trace)
	}

	t.Fatal(err)
}
