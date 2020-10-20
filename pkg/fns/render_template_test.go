package fns

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestTemplateRenderer_Filter(t *testing.T) {
	input := bytes.NewBufferString(`
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: v1
  kind: CustomResource
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/render-template: true
  foo:
    bar: '{{value "region"}}'
    baz:
    - '{{value "account-id"}}'
    - '{{value "domain-names" | sortAlpha | join ","}}'


functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: TemplateRenderer
  metadata:
    name: render-template
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-render-template:latest
  spec:
    kptfiles:
    - test-data/Kptfile
`)
	output := &bytes.Buffer{}

	config := RenderTemplateConfig{}
	resourceList := framework.ResourceList{
		Reader:         input,
		Writer:         output,
		FunctionConfig: &config,
	}

	if err := resourceList.Read(); err != nil {
		t.Fatal(err)
	}

	tokenReplacer := TemplateRenderer{Config: &config}
	for i := range resourceList.Items {
		if err := resourceList.Items[i].PipeE(&tokenReplacer); err != nil {
			t.Fatal(err)
		}
	}

	if err := resourceList.Write(); err != nil {
		t.Fatal(err)
	}

	expected := `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: v1
  kind: CustomResource
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/render-template: true
  foo:
    bar: 'ap-southeast-1'
    baz:
    - '111222333444'
    - 'dead.beef,example.com'

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: TemplateRenderer
  metadata:
    name: render-template
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-render-template:latest
  spec:
    kptfiles:
    - test-data/Kptfile
`

	if diff := cmp.Diff(normaliseYAML(expected), normaliseYAML(output.String())); diff != "" {
		t.Errorf("(-want +got)\n%s", diff)
	}
}

func normaliseYAML(doc string) string {
	return yaml.MustParse(doc).MustString()
}
