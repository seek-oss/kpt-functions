# Kpt Feature Flag Function (Proposed)

## Configuration

```yaml
apiVersion: kpt.seek.com/v1alpha1
kind: FeatureFlag
metadata:
  name: feature-flag
  annotations:
    config.kubernetes.io/function: |
      container:
        image: seek/kpt-feature-flag:latest
spec:
  flagDefinitions:
  - annotation: "kpt.seek.com/feature-flag/region"
    value: ap-southeast-2 # {"$kpt-set":"region"}
  - annotation: "kpt.seek.com/feature-flag/cluster"
    value: development-a # {"$kpt-set":"cluster-name"}
  - annotation: "kpt.seek.com/feature-flag/region-cluster"
    # This would be a substitution
    value: ap-southeast-2-development-a # {"$kpt-set":"region-cluster"}
```

To mark a resource as only being deployed in development-a, ap-southeast-2

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/feature-flag/region: ap-southeast-2
    kpt.seek.com/feature-flag/cluster: development-a
spec:
...
```

To mark a resource as only being deployed in development-a, across any region

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/feature-flag/cluster: development-a
spec:
...
```

To mark a resource as being in development-a across ap-southeast-1 and ap-southeast-2, but only in production-a in ap-southeast-1

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: example
  annotations:
    kpt.seek.com/feature-flag/region-cluster: ap-southeast-1-development-a, ap-southeast-2-development-a, ap-southeast-1-production-a
spec:
...
```

Syntax is `kpt.seek.com/feature-flag<free-text>: <match-values>`, where `<match-values>` is a comma separated list
of values. We will call files that have this annotation in them 'source files'.

When the function runs, any 'source files' will be treated as follows:

1. Find a matching entry for the annotation key in the `flagDefinitions.annotation` values.
2. If the corresponding `flagDefinitions.value` is in the list provided as the `<match-values>` as defined above, then
render the resource.

If the `flagDefinitions.value` is not included in the `<match-values>` then the resource is not rendered.

If the annotation is not found in the `flagDefinitions`, then an error is thrown.
