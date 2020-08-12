# Kpt Hash-Dependency Function

## Configuration

```yaml
apiVersion: kpt.seek.com/v1alpha1
kind: HashDependency
metadata:
  name: hash-dependency
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gantry-hash-dependency:latest
spec: {}
```

Example file

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
spec:
...
```

Syntax is `kpt.seek.com/hash-dependency<free-text>: <kind>/<name>`.
Replace `<free-text>` with something describing what the dependency represents.
This is optional, but is necessary when you want to hash multiple files, so that the annotation keys do not collide.
The reference resource must be in the same namespace as the annotated resource.

After adding hash

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
    ConfigMap/my-config-map: abcdef12345689
spec:
...
```
