# Basic sync example

This example shows off the functionality of the sync function.

The [cluster](./cluster) directory contains the cluster configuration.
It includes some variables and a template.

The [packages](./packages) directory contains the package configuration.
It consists of a single Kpt package, with a number of setters.

## Usage

```bash
kpt fn source \
  examples/sync/basic/cluster/packages.yaml \
  | kpt fn run \
  --image docker.io/seek/kpt-sync:latest \
  --network -- logLevel=debug \
  | kpt fn sink .
```

This will cause the package in the [packages](./packages) directory to be rendered using the values defined in
[cluster/packages.yaml](cluster/packages.yaml).

Of interest:
* the value of `replicas` is taken from the more specific definition in the package, rather than the global defintion.
* variables at the package level need not exist at the global/cluster level. This means that packages can have the same
setter defined, and values for that setter can be set independently.
* The template in the config map is rendered.

You may need to manually remove the files from the `cluster/packages` directory before syncing again.
