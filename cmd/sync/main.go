package main

import (
	"io/ioutil"
	"os"
	"strconv"

	"sigs.k8s.io/kustomize/kyaml/errors"

	v1 "k8s.io/api/core/v1"

	"github.com/seek-oss/kpt-functions/pkg/filters"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"

	"github.com/rs/zerolog"
)

const (
	logLevelFunctionArg  = "logLevel"
	logLevelDefaultValue = zerolog.InfoLevel

	deleteCacheFunctionArg  = "deleteCache"
	deleteCacheDefaultValue = true

	cacheDirFunctionArg = "cacheDir"
)

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func main() {
	proc := newProcessor()
	rw := phonyByteReadWriter()
	//rw := realByteReadWriter()
	if err := framework.Execute(proc, rw); err != nil {
		logger.Fatal().Err(err).Msgf("Error performing sync operation %v", err == nil)
	}
}

func newProcessor() framework.ResourceListProcessor {
	var functionConfig v1.ConfigMap
	delegate := &filters.ClusterPackagesFilter{Logger: logger}

	filter := kio.FilterFunc(func(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
		var err error
		var ok bool

		logLevel := logLevelDefaultValue
		if v, ok := functionConfig.Data[logLevelFunctionArg]; ok {
			logLevel, err = zerolog.ParseLevel(v)
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not parse log level")
			}
		}

		zerolog.SetGlobalLevel(logLevel)

		cleanCache := deleteCacheDefaultValue

		delegate.CacheDir, ok = functionConfig.Data[cacheDirFunctionArg]
		if ok {
			cleanCache = false
		} else {
			delegate.CacheDir, err = ioutil.TempDir("", "")
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not create temporary cache directory")
			}
		}

		if v, ok := functionConfig.Data[deleteCacheFunctionArg]; ok {
			cleanCache, err = strconv.ParseBool(v)
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not parse deleteCache argument")
			}
		}

		defer func() {
			if !cleanCache {
				return
			}

			if err := os.RemoveAll(delegate.CacheDir); err != nil {
				logger.Fatal().Err(err).Msgf("Could not delete cache directory %s", delegate.CacheDir)
			}
		}()

		return delegate.Filter(nodes)
	})

	return framework.SimpleProcessor{Config: &functionConfig, Filter: filter}
}

func realByteReadWriter() *kio.ByteReadWriter {
	return &kio.ByteReadWriter{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
}

func phonyByteReadWriter() *kio.ByteReadWriter {
	f, err := os.Open("cmd/sync/test-data/kpt-rl.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msgf("Error reading input file")
	}

	return &kio.ByteReadWriter{
		Reader: f,
		Writer: os.Stdout,
		FunctionConfig: kyaml.MustParse(
			`
data:
  logLevel: debug
      `,
		),
	}
}
