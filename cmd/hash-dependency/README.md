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
        image: seek/kpt-hash-dependency:latest
spec: {}
```

Example file

```yaml
apiVersion: apps/v1
kind: StatefulSet
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
kind: StatefulSet
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
    ConfigMap/my-config-map: abcdef12345689
spec:
...
```

Hashing also works inside of a PodSpec in Deployments and DaemonSets, for example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: example
spec:
  template:
    metadata:
      annotations:
        kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
...
```

will hash to:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: example
spec:
  template:
    metadata:
      annotations:
        kpt.seek.com/hash-dependency/config-map: ConfigMap/my-config-map
        ConfigMap/my-config-map: abcdef12345689
...
```

This allows you to have a pod as part of a deployment that depends on another resource, with the pods being updated
when the hash of the dependant resource changes.
