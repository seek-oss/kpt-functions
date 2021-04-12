# Basic sync example

This example shows off the functionality of the sync function.

## Usage

```bash
kpt fn source \
  examples/sync/basic/cluster/packages.yaml \
  | kpt fn run \
  --image docker.io/seek/kpt-sync:latest \
  --network -- logLevel=debug \
  | kpt fn sink .
```
