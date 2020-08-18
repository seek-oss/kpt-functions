# Kpt Functions

This repository provides [Kpt functions](https://googlecontainertools.github.io/kpt/guides/producer/functions)
that are used to extend Kpt's functionality.

## Function library

### hash-dependency

A common Kubernetes pattern is to have a workload resource (`Deployment`, `StatefulSet`, `DaemonSet` etc.) that mounts
a `ConfigMap` resource which contains configuration. Frequently, we need to force this workload to restart in order
for the config changes to take effect. A nice way of achieving this is to cause a change in the configuration resource
to trigger a change in the workload resource's definition, which causes Kubernetes to replace the workload, forcing it
to reload config.

This function allows you to designate resources as 'dependencies' via an annotation. When the function is executed,
the contents of the 'dependency' is hashed, with this hash added as an annotation. When the contents of the 'dependency'
change, so will the hash, triggering a workload replacement.

See the [README](cmd/hash-dependency/README.md) for usage instructions.

### token-replace

Application authors attempting to adapt existing software to Kubernetes may not have gone to the effort of creating a
CRD to configure their application. Usually, this manifests as some sort of bespoke configuration file that is expected
to be passed to the application via a mounted `ConfigMap`. This doesn't work well with `kpt`, because `kpt` is incapable
of setting values inside of a YAML literal, beacuse there is no way to escape the YAML literal in order to place the
required `# {"kpt-set":"my-variable"}` comment.

This function allows targeted token replacement with in a `ConfigMap`, in order to address this common use-case.

See the [README](cmd/token-replace/README.md) for usage instructions.

## Repo layout

To create a new function, create a new subfolder under the [cmd]() directory. Subfolders of this directory will be
discovered automatically by the Makefile eg. for building or publishing Docker images.

## Make targets

### `make test`

Runs unit tests

### `make build-<function-name>`

Builds a Docker image out of the function at `cmd/<function-name>`, tagged as `seek/kpt-<function-name>:$(VERSION)`.
If the `VERSION` variable is unset, its value defaults to `latest`.

### `make build-all`

Runs `make build-<function-name>` against all functions found in the [cmd]() directory.

### `make publish-<function-name>`

Pushes the Docker image build by `make build-<function-name>`.

### `make publish-all`

Runs `make publish-<function-name>` against all functions found in the [cmd]() directory.
