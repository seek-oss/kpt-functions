package fns

import (
  "bytes"
  "github.com/go-errors/errors"
  "testing"

  "github.com/google/go-cmp/cmp"
  "sigs.k8s.io/kustomize/kyaml/fn/framework"
  "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestTemplateRenderer_Simple_Filter(t *testing.T) {
	input := bytes.NewBufferString(`
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: v1
  kind: CustomResource
  metadata:
    name: example1
    namespace: example
    annotations:
      kpt.seek.com/render-template: true
  spec:
    foo:
      bar: '{{value "region"}}'
      baz:
      - '{{value "account-id"}}'
      - '{{value "domain-names" | sortAlpha | join ","}}'

- apiVersion: v1
  kind: AnotherCustomResource
  metadata:
    name: example2
    namespace: example
    annotations:
      kpt.seek.com/render-template: true
      kpt.seek.com/render-template/delimiters: "[[ ]]"
  spec:
    foo:
      bar: '[[value "region"]]'

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: RenderTemplate
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
      fatalError(t, err)
		}
	}

	if err := resourceList.Write(); err != nil {
    fatalError(t, err)
	}

	expected := `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: v1
  kind: CustomResource
  metadata:
    name: example1
    namespace: example
    annotations:
      kpt.seek.com/render-template: true
  spec:
    foo:
      bar: 'ap-southeast-1'
      baz:
      - '111222333444'
      - 'dead.beef,example.com'

- apiVersion: v1
  kind: AnotherCustomResource
  metadata:
    name: example2
    namespace: example
    annotations:
      kpt.seek.com/render-template: true
      kpt.seek.com/render-template/delimiters: "[[ ]]"
  spec:
    foo:
      bar: 'ap-southeast-1'

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: RenderTemplate
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

func TestTemplateRenderer_SubTemplate_Filter(t *testing.T) {
  input := bytes.NewBufferString(`
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: identity.aws.crossplane.io/v1beta1
  kind: IAMRole
  metadata:
    name: cert-manager-role # {"$kpt-set":"role-name"}
    namespace: cert-manager
    annotations:
      kpt.seek.com/render-template: true
  spec:
    forProvider:
      assumeRolePolicyDocument: |-
        {{render "irsa-policy" "cert-manager" "cert-manager"}}
    reclaimPolicy: Delete
    providerRef:
      name: aws-provider

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: RenderTemplate
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
      fatalError(t, err)
    }
  }

  if err := resourceList.Write(); err != nil {
    fatalError(t, err)
  }

  expected := `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: identity.aws.crossplane.io/v1beta1
  kind: IAMRole
  metadata:
    name: cert-manager-role # {"$kpt-set":"role-name"}
    namespace: cert-manager
    annotations:
      kpt.seek.com/render-template: true
  spec:
    forProvider:
      assumeRolePolicyDocument: |
        {
          "Version": "2012-10-17",
          "Statement": [
            {
              "Effect": "Allow",
              "Principal": {
                "Federated": 'arn:aws:iam::111222333444:oidc-provider/oidc.eks.ap-southeast-1.amazonaws.com/id/ABCDEFG'
              },
              "Action": "sts:AssumeRoleWithWebIdentity",
              "Condition": {
                "StringEquals": {
                  "oidc.eks.ap-southeast-1.amazonaws.com/id/ABCDEFG:sub": "system:serviceaccount:cert-manager:cert-manager"
                }
              }
            }
          ]
        }
    reclaimPolicy: Delete
    providerRef:
      name: aws-provider

functionConfig:
  apiVersion: kpt.seek.com/v1alpha1
  kind: RenderTemplate
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

func fatalError(t *testing.T, err error) {
  t.Helper()

  if e, ok := err.(*errors.Error); ok {
    trace := e.ErrorStack()
    t.Fatal(err, trace)
  }

  t.Fatal(err)
}