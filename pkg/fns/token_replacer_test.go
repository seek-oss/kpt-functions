package fns

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"testing"
)

func TestTokenReplacer_ConfigMap_Filter(t *testing.T) {
	input := bytes.NewBufferString(`
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/token-replace: enabled
  data:
    template.tmpl: |
      {"Region":"$region","Cluster":"$cluster","Status":"{{ .Status }}","Alerts":[{{- range $index, $alert := .Alerts -}}{{ if ne $index 0 }},{{ end }}{"AlarmName":"{{- if eq .Labels.severity "warning" -}}[{{ .Labels.severity | str_UpperCase -}}:P3] {{ .Labels.alertname -}}",{{- else if eq .Labels.severity "critical" -}}[{{ .Labels.severity | str_UpperCase -}}:P1] {{ .Labels.alertname -}}",{{- end }}"AlarmDescription":"{{ .Annotations.message }}","Runbook":"{{- .Annotations.runbook_url -}}",{{- $length := len .Labels -}}{{- if ne $length 0 -}}{{- range $key,$val := .Labels -}}{{- if ne $key "alertname" }}"{{- $key | str_Title }}":"{{ $val -}}",{{- end -}}{{- end -}}{{- end }}"StartsAt":"{{ .StartsAt }}","EndsAt":"{{ .EndsAt }}","GeneratorURL":"{{ .GeneratorURL }}"}{{- end }}],"ExternalURL":"{{ .ExternalURL }}","Version":"{{ .Version }}"}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: TokenReplacer
  metadata:
    name: token-replace
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-token-replace:latest
  spec:
    replacements:
    - token: "$region"
      value: ap-southeast-1
    - token: "$cluster"
      value: development-a
`)
	output := &bytes.Buffer{}

	config := TokenReplaceConfig{}
	resourceList := framework.ResourceList{
		Reader:         input,
		Writer:         output,
		FunctionConfig: &config,
	}

	if err := resourceList.Read(); err != nil {
		t.Fatal(err)
	}

	tokenReplacer := TokenReplacer{Config: &config}
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
  kind: ConfigMap
  metadata:
    name: example
    namespace: example
    annotations:
      kpt.seek.com/token-replace: enabled
      kpt.seek.com/token-replace-nodes:
  data:
    template.tmpl: |
      {"Region":"ap-southeast-1","Cluster":"development-a","Status":"{{ .Status }}","Alerts":[{{- range $index, $alert := .Alerts -}}{{ if ne $index 0 }},{{ end }}{"AlarmName":"{{- if eq .Labels.severity "warning" -}}[{{ .Labels.severity | str_UpperCase -}}:P3] {{ .Labels.alertname -}}",{{- else if eq .Labels.severity "critical" -}}[{{ .Labels.severity | str_UpperCase -}}:P1] {{ .Labels.alertname -}}",{{- end }}"AlarmDescription":"{{ .Annotations.message }}","Runbook":"{{- .Annotations.runbook_url -}}",{{- $length := len .Labels -}}{{- if ne $length 0 -}}{{- range $key,$val := .Labels -}}{{- if ne $key "alertname" }}"{{- $key | str_Title }}":"{{ $val -}}",{{- end -}}{{- end -}}{{- end }}"StartsAt":"{{ .StartsAt }}","EndsAt":"{{ .EndsAt }}","GeneratorURL":"{{ .GeneratorURL }}"}{{- end }}],"ExternalURL":"{{ .ExternalURL }}","Version":"{{ .Version }}"}

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: TokenReplacer
  metadata:
    name: token-replace
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-token-replace:latest
  spec:
    replacements:
    - token: "$region"
      value: ap-southeast-1
    - token: "$cluster"
      value: development-a
`

	if diff := cmp.Diff(normaliseYAML(expected), normaliseYAML(output.String())); diff != "" {
		t.Errorf("(-want +got)\n%s", diff)
	}
}

func TestTokenReplacer_NonConfigMap_Filter(t *testing.T) {
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
      kpt.seek.com/token-replace: enabled
  foo:
    bar: "$region"
    baz:
    - "$cluster"


functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: TokenReplacer
  metadata:
    name: token-replace
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-token-replace:latest
  spec:
    replacements:
    - token: "$region"
      value: ap-southeast-1
    - token: "$cluster"
      value: development-a
`)
  output := &bytes.Buffer{}

  config := TokenReplaceConfig{}
  resourceList := framework.ResourceList{
    Reader:         input,
    Writer:         output,
    FunctionConfig: &config,
  }

  if err := resourceList.Read(); err != nil {
    t.Fatal(err)
  }

  tokenReplacer := TokenReplacer{Config: &config}
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
      kpt.seek.com/token-replace: enabled
  foo:
    bar: "ap-southeast-1"
    baz:
    - "development-a"

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: TokenReplacer
  metadata:
    name: token-replace
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gantry-token-replace:latest
  spec:
    replacements:
    - token: "$region"
      value: ap-southeast-1
    - token: "$cluster"
      value: development-a
`

  if diff := cmp.Diff(normaliseYAML(expected), normaliseYAML(output.String())); diff != "" {
    t.Errorf("(-want +got)\n%s", diff)
  }
}

func normaliseYAML(doc string) string {
	return yaml.MustParse(doc).MustString()
}
