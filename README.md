# Kpt Functions

This repository provides [Kpt functions](https://googlecontainertools.github.io/kpt/guides/producer/functions)
that are used to extend Kpt's functionality.

## Intended Usage of the Sync Function

Suppose that the target repository has the following structure.

```
.
├── config
    ├── development
    │   └── ap-southeast-2
    │       └── a
    │           └── packages.yaml
    └── production
        └── ap-southeast-2
            └── a
                └── packages.yaml
```

Below shows what `config/development/ap-southeast-2/a/packages.yaml` looks like.

```yaml
apiVersion: kpt.seek.com/v1alpha1
kind: ClusterPackages
metadata:
  name: ap-southeast-2-development-a
spec:
  baseDir: config/development/ap-southeast-2/a
  packages:
  - name: cert-manager
    git:
      repo: git@github.com:seek-oss/packages.git
      directory: cert-manager
      ref: 5fc702d3dd0f46509283cb0bcc4a3327d1ee8b1d
      # Packages may define their own overrides
      #variables:
      #- name: foo
      #  value: bar
  - name: istio-system
    git:
      repo: git@github.com:seek-oss/packages.git
      directory: istio-system
      ref: 5fc702d3dd0f46509283cb0bcc4a3327d1ee8b1d
  variables:
  - name: account-id
    value: "1234"
  - name: region
    value: ap-southeast-2
  - name: region-short
    value: apse2
  - name: cluster
    value: development-a
  - name: environment
    value: development
  - name: oidc-provider-id
    value: ABCDEFG
  - name: internal-domain-name
    value: ap-southeast-2.development.internal.seek.com
  - name: external-domain-name
    value: ap-southeast-2.development.external.seek.com
  - name: internal-hosted-zone-id
    value: INT001
  - name: external-hosted-zone-id
    value: EXT001
  - name: irsa-policy
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
```

Then the following operation would sync all packages.

```bash
kpt fn source \
  config/development/ap-southeast-2/a/packages.yaml \
  config/production/ap-southeast-2/a/packages.yaml \
  | kpt fn run \
  --image docker.io/seek/kpt-sync:latest \
  --mount type=bind,source="${HOME}/.ssh/id_rsa,target=/.ssh/id_rsa,readonly" \
  --network -- logLevel=debug \
  | kpt fn sink .
```

