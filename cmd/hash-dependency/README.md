# Kpt Hash-Dependency Function

## Motivations

Some Kubernetes applications read their config from a config map or some other configuration source, but are not
configured to automatically reload this configuration when it changes. The solution to this is to restart the
workload, but this must be done manually.

The hash dependency function allows for embedding a hash of a dependant piece of configuration in a workload spec.
When the configuration changes, the workload will necessarily be re-created because of the changed
annotation in its spec.

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
