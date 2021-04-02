package filters

import (
	"bytes"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/kio"

	"github.com/go-errors/errors"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func newProcessor() framework.ResourceListProcessor {
	renderer := &TemplateFilter{}
	return framework.SimpleProcessor{
		Filter: renderer,
	}
}

func TestTemplateRenderer_Simple_Filter(t *testing.T) {
	input := bytes.NewBufferString(`
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1alpha1
  kind: Kptfile
  metadata:
    name: test
  openAPI:
    definitions:
      io.k8s.cli.setters.account-id:
        type: string
        x-k8s-cli:
          setter:
            name: account-id
            value: 111222333444
      io.k8s.cli.setters.region:
        type: string
        x-k8s-cli:
          setter:
            name: region
            value: ap-southeast-1
      io.k8s.cli.setters.domain-names:
        type: array
        items:
          type: string
        x-k8s-cli:
          setter:
            name: domain-names
            listValues:
            - "example.com"
            - "dead.beef"
      io.k8s.cli.setters.oidc-provider-id:
        type: string
        x-k8s-cli:
          setter:
            name: oidc-provider-id
            value: ABCDEFG
      io.k8s.cli.setters.irsa-policy:
        type: string
        x-k8s-cli:
          setter:
            name: irsa-policy
            value: |
              {
                "Version": "2012-10-17",
                "Statement": [
                  {
                    "Effect": "Allow",
                    "Principal": {
                      "Federated": 'arn:aws:iam::{{value "account-id"}}:oidc-provider/oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}'
                    },
                    "Action": "sts:AssumeRoleWithWebIdentity",
                    "Condition": {
                      "StringEquals": {
                        "oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}:sub": "system:serviceaccount:{{args 0}}:{{args 1}}"
                      }
                    }
                  }
                ]
              }

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
`)

	output := &bytes.Buffer{}

	rw := &kio.ByteReadWriter{
		Reader: input,
		Writer: output,
	}

	if err := framework.Execute(newProcessor(), rw); err != nil {
		fatalError(t, err)
	}

	expected := `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1alpha1
  kind: Kptfile
  metadata:
    name: test
  openAPI:
    definitions:
      io.k8s.cli.setters.account-id:
        type: string
        x-k8s-cli:
          setter:
            name: account-id
            value: 111222333444
      io.k8s.cli.setters.region:
        type: string
        x-k8s-cli:
          setter:
            name: region
            value: ap-southeast-1
      io.k8s.cli.setters.domain-names:
        type: array
        items:
          type: string
        x-k8s-cli:
          setter:
            name: domain-names
            listValues:
            - "example.com"
            - "dead.beef"
      io.k8s.cli.setters.oidc-provider-id:
        type: string
        x-k8s-cli:
          setter:
            name: oidc-provider-id
            value: ABCDEFG
      io.k8s.cli.setters.irsa-policy:
        type: string
        x-k8s-cli:
          setter:
            name: irsa-policy
            value: |
              {
                "Version": "2012-10-17",
                "Statement": [
                  {
                    "Effect": "Allow",
                    "Principal": {
                      "Federated": 'arn:aws:iam::{{value "account-id"}}:oidc-provider/oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}'
                    },
                    "Action": "sts:AssumeRoleWithWebIdentity",
                    "Condition": {
                      "StringEquals": {
                        "oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}:sub": "system:serviceaccount:{{args 0}}:{{args 1}}"
                      }
                    }
                  }
                ]
              }

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
- apiVersion: kpt.dev/v1alpha1
  kind: Kptfile
  metadata:
    name: test
  openAPI:
    definitions:
      io.k8s.cli.setters.account-id:
        type: string
        x-k8s-cli:
          setter:
            name: account-id
            value: 111222333444
      io.k8s.cli.setters.region:
        type: string
        x-k8s-cli:
          setter:
            name: region
            value: ap-southeast-1
      io.k8s.cli.setters.domain-names:
        type: array
        items:
          type: string
        x-k8s-cli:
          setter:
            name: domain-names
            listValues:
            - "example.com"
            - "dead.beef"
      io.k8s.cli.setters.oidc-provider-id:
        type: string
        x-k8s-cli:
          setter:
            name: oidc-provider-id
            value: ABCDEFG
      io.k8s.cli.setters.irsa-policy:
        type: string
        x-k8s-cli:
          setter:
            name: irsa-policy
            value: |
              {
                "Version": "2012-10-17",
                "Statement": [
                  {
                    "Effect": "Allow",
                    "Principal": {
                      "Federated": 'arn:aws:iam::{{value "account-id"}}:oidc-provider/oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}'
                    },
                    "Action": "sts:AssumeRoleWithWebIdentity",
                    "Condition": {
                      "StringEquals": {
                        "oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}:sub": "system:serviceaccount:{{args 0}}:{{args 1}}"
                      }
                    }
                  }
                ]
              }

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
`)

	output := &bytes.Buffer{}

	rw := &kio.ByteReadWriter{
		Reader: input,
		Writer: output,
	}

	if err := framework.Execute(newProcessor(), rw); err != nil {
		fatalError(t, err)
	}

	expected := `
apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1alpha1
  kind: Kptfile
  metadata:
    name: test
  openAPI:
    definitions:
      io.k8s.cli.setters.account-id:
        type: string
        x-k8s-cli:
          setter:
            name: account-id
            value: 111222333444
      io.k8s.cli.setters.region:
        type: string
        x-k8s-cli:
          setter:
            name: region
            value: ap-southeast-1
      io.k8s.cli.setters.domain-names:
        type: array
        items:
          type: string
        x-k8s-cli:
          setter:
            name: domain-names
            listValues:
            - "example.com"
            - "dead.beef"
      io.k8s.cli.setters.oidc-provider-id:
        type: string
        x-k8s-cli:
          setter:
            name: oidc-provider-id
            value: ABCDEFG
      io.k8s.cli.setters.irsa-policy:
        type: string
        x-k8s-cli:
          setter:
            name: irsa-policy
            value: |
              {
                "Version": "2012-10-17",
                "Statement": [
                  {
                    "Effect": "Allow",
                    "Principal": {
                      "Federated": 'arn:aws:iam::{{value "account-id"}}:oidc-provider/oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}'
                    },
                    "Action": "sts:AssumeRoleWithWebIdentity",
                    "Condition": {
                      "StringEquals": {
                        "oidc.eks.{{value "region"}}.amazonaws.com/id/{{value "oidc-provider-id"}}:sub": "system:serviceaccount:{{args 0}}:{{args 1}}"
                      }
                    }
                  }
                ]
              }

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
