package main

import (
	"io"
	"os"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"sigs.k8s.io/kustomize/kyaml/kio"

	"github.com/rs/zerolog"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	proc := NewSyncProcessor()
	rw := phonyByteReadWriter()
	//rw := realByteReadWriter()
	if err := framework.Execute(proc, rw); err != nil {
		logger.Fatal().Err(err).Msgf("Error performing sync operation %v", err == nil)
	}
}

func realByteReadWriter() *kio.ByteReadWriter {
	return &kio.ByteReadWriter{
		Reader: os.Stdin,
		Writer: io.Discard,
	}
}

func phonyByteReadWriter() *kio.ByteReadWriter {
	f, err := os.Open("cmd/e2e/test-data/kpt-rl.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msgf("Error reading input file")
	}

	return &kio.ByteReadWriter{
		Reader: f,
		Writer: io.Discard,
		FunctionConfig: kyaml.MustParse(
			`
data:
  baseDir: foobar
      `,
		),
	}
}
