# SEEK-OSS Kpt functions

Opinionated [Kpt](https://googlecontainertools.github.io/kpt/) [functions](https://googlecontainertools.github.io/kpt/guides/producer/functions/)
in use at Seek.

These extend the functionality of Kpt to add features that are currently missing or domain specific.

🚧 Note: these are under heavy development

## Current function library

[`kpt-sync`](./cmd/sync/README.md): A function to declaratively sync multiple Kpt packages that share configuration.

[`kpt-hash-dependency`](./cmd/hash-dependency/README.md): A function to force updates to a resource based on the hash of another resource changing.

## License

MIT
