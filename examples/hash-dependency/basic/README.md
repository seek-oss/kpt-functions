# Basic hash-dependency example

This is a basic example of the hash-dependency function.

In this example, we have a deployment which depends on a config map.
The config map's value is hashed, and inserted into the annotations of the deployment's pods.
This should force the pod to be updated when the config map's data changes.

## Usage

To populate the deployment's pod annotations with an initial hash, run:

```bash
kpt fn source examples/hash-dependency/basic/input \
  | kpt fn run --image docker.io/seek/kpt-hash-dependency:latest -- logLevel=debug \
  | kpt fn sink examples/hash-dependency/basic/output
```

Note that there is now a new annotation in the pod spec:

```
...
  template:
    metadata:
      annotations:
        kpt.seek.com/hash-dependency/config-map: ConfigMap/config-map
        ConfigMap/config-map: '0939f7d83cee21dd610882b0174b074c718123316de08c5e5da0547660e17c88'
...
```

Modify the contents of the input config map, and then re-run the command above.
The value of the `ConfigMap/config-map` annotation should change to reflect the new hash.
