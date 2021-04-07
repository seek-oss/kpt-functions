# Kpt Hash-Dependency Function

## Configuration

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

## Usage

```bash
kpt fn source <dir-or-files> \
  | kpt fun run --image docker.io/seek/kpt-hash-dependency:latest -- logLevel=debug
  | kpt fn sink <dir>
```
