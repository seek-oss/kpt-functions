package main

import (
	"os"

	"github.com/seek-oss/kpt-functions/pkg/filters"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"

	"github.com/rs/zerolog"
)

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	proc := newProcessor()
	rw := phonyByteReadWriter()
	//rw := realByteReadWriter()
	if err := framework.Execute(proc, rw); err != nil {
		logger.Fatal().Err(err).Msgf("Error performing sync operation %v", err == nil)
	}
}

func newProcessor() framework.ResourceListProcessor {
	config := &filters.ClusterPackagesFilterConfig{
		Data: struct {
			LogLevel string `yaml:"logLevel,omitempty"`
		}{
			LogLevel: "",
		},
	}
	filter := &filters.ClusterPackagesFilter{
		Config: config,
		Logger: logger,
	}

	return framework.SimpleProcessor{
		Config: config,
		Filter: filter,
	}
}

func realByteReadWriter() *kio.ByteReadWriter {
	return &kio.ByteReadWriter{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
}

func phonyByteReadWriter() *kio.ByteReadWriter {
	f, err := os.Open("cmd/e2e/test-data/kpt-rl.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msgf("Error reading input file")
	}

	return &kio.ByteReadWriter{
		Reader: f,
		Writer: os.Stdout,
		FunctionConfig: kyaml.MustParse(
			`
data:
  baseDir: target/packages
      `,
		),
	}
}
