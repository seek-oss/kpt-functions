# Kpt Token-Replace Function

## Configuration

```yaml
apiVersion: kpt.seek.com/v1alpha1
kind: TokenReplace
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
```

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/token-replace: enabled
data:
  template.tmpl: |
    {"Region":"$region","Cluster":"$cluster","Status":"{{ .Status }}","Alerts":[{{- range $index, $alert := .Alerts -}}{{ if ne $index 0 }},{{ end }}{"AlarmName":"{{- if eq .Labels.severity "warning" -}}[{{ .Labels.severity | str_UpperCase -}}:P3] {{ .Labels.alertname -}}",{{- else if eq .Labels.severity "critical" -}}[{{ .Labels.severity | str_UpperCase -}}:P1] {{ .Labels.alertname -}}",{{- end }}"AlarmDescription":"{{ .Annotations.message }}","Runbook":"{{- .Annotations.runbook_url -}}",{{- $length := len .Labels -}}{{- if ne $length 0 -}}{{- range $key,$val := .Labels -}}{{- if ne $key "alertname" }}"{{- $key | str_Title }}":"{{ $val -}}",{{- end -}}{{- end -}}{{- end }}"StartsAt":"{{ .StartsAt }}","EndsAt":"{{ .EndsAt }}","GeneratorURL":"{{ .GeneratorURL }}"}{{- end }}],"ExternalURL":"{{ .ExternalURL }}","Version":"{{ .Version }}"}
```
