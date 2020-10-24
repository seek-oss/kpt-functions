# Kpt Token-Replace Function

## Configuration

```yaml
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
  - Kptfile
```

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/render-template: true
data:
  file.yaml: |
    region: {{value "region"}}
    accountID: {{value "account-id"}}
```
