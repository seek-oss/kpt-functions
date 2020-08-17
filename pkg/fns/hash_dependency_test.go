package fns

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"testing"
)

type TestCase struct {
	testCase       string
	input          string
	expectedOutput string
}

func TestHashDependency_Filter(t *testing.T) {
	testCases := []TestCase{
		{
			testCase: "Basic replacement",
			input: `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/hash-dependency: ConfigMap/my-config-map
  spec: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
			expectedOutput: `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/hash-dependency: ConfigMap/my-config-map
      ConfigMap/my-config-map: 'dfa6c3c082ad3ee44f29b13328af93f4c00e9438e93f7c8b5a58dd389cd491e6'
  spec: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
		},
		{
			testCase: "Hash target not found",
			input: `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/hash-dependency/config-map: ConfigMap/config-map-not-there
  spec: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
			expectedOutput: `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/hash-dependency/config-map: ConfigMap/config-map-not-there
  spec: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
		},
		{
			testCase: "Recomputes hash when label is already there",
			input: `
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
      ConfigMap/my-config-map: 'abc134'
  spec: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
			expectedOutput: `
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
      ConfigMap/my-config-map: 'dfa6c3c082ad3ee44f29b13328af93f4c00e9438e93f7c8b5a58dd389cd491e6'
  spec: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-config-map
    namespace: example
  data: {}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
		},
		{
			testCase: "Hashes multiple targets",
			input: `
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
      kpt.seek.com/hash-dependency/another-type: AnotherType/another-type
  spec: {}
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

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
			expectedOutput: `
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
      kpt.seek.com/hash-dependency/another-type: AnotherType/another-type
      ConfigMap/my-config-map: 'dfa6c3c082ad3ee44f29b13328af93f4c00e9438e93f7c8b5a58dd389cd491e6'
      AnotherType/another-type: '86db829e5f05670ba1162010566a09090bedd562d9f7b95dd94cb98447978f3a'
  spec: {}
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

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: HashDependency
  metadata:
    name: hash-dependency
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-hash-dependency:latest
  spec: {}
`,
		},
	}

	for index := range testCases {
		testCase := testCases[index]

		output := &bytes.Buffer{}

		config := HashDependencyConfig{}
		resourceList := framework.ResourceList{
			Reader:         bytes.NewBufferString(testCase.input),
			Writer:         output,
			FunctionConfig: &config,
		}
		if err := resourceList.Read(); err != nil {
			t.Fatal(err)
		}

		dependencyHasher := DependencyHasher{ResourceListItems: resourceList.Items}
		for i := range resourceList.Items {
			if err := resourceList.Items[i].PipeE(&dependencyHasher); err != nil {
				t.Fatal(err)
			}
		}

		if err := resourceList.Write(); err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(normaliseYAML(testCase.expectedOutput), normaliseYAML(output.String())); diff != "" {
			t.Errorf("Test case failed: %s\n(-want +got)\n%s", testCase.testCase, diff)
		}

	}
}
