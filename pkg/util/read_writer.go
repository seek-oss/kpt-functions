package util

import (
  "os"
  "sigs.k8s.io/kustomize/kyaml/errors"
  "sigs.k8s.io/kustomize/kyaml/kio"
)

// readWriter returns a kio.ByteReadWriter that is configured to read from stdin if no command line argument
// has been specified. If command line arguments are specified, the first argument is assumed to be a file
// containing a framework.ResourceList - this can be useful for debugging locally.
func ReadWriter() (*kio.ByteReadWriter, error) {
  r := os.Stdin
  if len(os.Args) > 1 {
    var err error
    r, err = os.Open(os.Args[1])
    if err != nil {
      return nil, errors.WrapPrefixf(err, "could not read file argument")
    }
  }

  return &kio.ByteReadWriter{Reader: r}, nil
}
