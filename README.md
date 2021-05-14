# SEEK-OSS Kpt functions

Opinionated [Kpt](https://googlecontainertools.github.io/kpt/) [functions](https://googlecontainertools.github.io/kpt/guides/producer/functions/)
in use at Seek.

These extend the functionality of Kpt to add features that are currently missing or domain specific.

If you're new to Kpt, we recommend following our [Kpt tutorial](./examples/tutorial/README.md).
This tutorial will take you through the basic Kpt workflows, which will serve as a basis for understanding the utility
of the functions in this repo.

🚧 Note: these are under heavy development

## Current function library

[`kpt-sync`](./cmd/sync/README.md): A function to declaratively sync multiple Kpt packages that share configuration.

[`kpt-hash-dependency`](./cmd/hash-dependency/README.md): A function to force updates to a resource based on the hash of another resource changing.

## Releasing

Releasing a function in this repo means building and pushing a Docker image that contains the function.

To perform a release, create a Github release with a tag of the format `<function-name>/v<semver-version>`.
For example, to release version `1.2.3` of the `sync` function, tag with `sync/v1.2.3`.
This will start a Github actions build that will build and push to Dockerhub as `seek/kpt-sync:1.2.3`.

## License

MIT
