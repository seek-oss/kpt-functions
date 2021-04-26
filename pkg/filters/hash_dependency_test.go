package filters

import (
  "bytes"
  "sigs.k8s.io/kustomize/kyaml/fn/framework"
  "sigs.k8s.io/kustomize/kyaml/fn/framework/frameworktestutil"
  "sigs.k8s.io/kustomize/kyaml/kio"
  "testing"
)

func TestHashDependencyFilter(t *testing.T) {
	newProcessor := func() framework.ResourceListProcessor {
		return framework.SimpleProcessor{
			Filter: &HashDependencyFilter{},
		}

	}
	checker := frameworktestutil.ProcessorResultsChecker{
		Processor:         newProcessor,
		TestDataDirectory: "hash_dependency_testdata",
	}

	checker.Assert(t)
}

func BenchmarkHashDependencyFilter(b *testing.B) {
	input := `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
  spec:
    template:
      metadata:
        annotations:
          kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}
- apiVersion: custom-namespace.seek.com/v1
  kind: AnotherType
  metadata:
    name: another-type
    namespace: example
  data: {}
`
	output := &bytes.Buffer{}
	rw := &kio.ByteReadWriter{
		Reader: bytes.NewBufferString(input),
		Writer: output,
	}

	filter := &HashDependencyFilter{}

	for n := 0; n < b.N; n++ {
		_ = framework.Execute(newProcessor(filter), rw)
	}
}
