package main

import (
  "github.com/seek-oss/kpt-functions/pkg/log"
  "github.com/seek-oss/kpt-functions/pkg/util"
  "sigs.k8s.io/kustomize/kyaml/errors"

  v1 "k8s.io/api/core/v1"

  "github.com/seek-oss/kpt-functions/pkg/filters"

  kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

  "sigs.k8s.io/kustomize/kyaml/fn/framework"
  "sigs.k8s.io/kustomize/kyaml/kio"

  "github.com/rs/zerolog"
)

const (
	logLevelFunctionArg     = "logLevel"

	defaultLogLevel   = zerolog.InfoLevel
)

// logger is the configured zerolog Logger instance.
var logger zerolog.Logger

// Entry point for the sync custom Kpt function.
func main() {
  logger = log.GetLogger(defaultLogLevel)
	if err := realMain(); err != nil {
		logger.Fatal().Err(err).Msgf("Error hashing dependencies")
	}
}

// realMain executes the sync operation and returns any errors.
func realMain() error {
	proc := newProcessor()
	rw, err := util.ReadWriter()
	if err != nil {
		return err
	}

	return framework.Execute(proc, rw)
}

// newProcessor returns the framework.ResourceListProcessor for the custom sync function.
func newProcessor() framework.ResourceListProcessor {
	var cm v1.ConfigMap
	delegate := &filters.HashDependencyFilter{Logger: logger}

	filter := kio.FilterFunc(func(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
		var err error

		logLevel := defaultLogLevel
		if v, ok := cm.Data[logLevelFunctionArg]; ok {
			logLevel, err = zerolog.ParseLevel(v)
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not parse log level")
			}
		}

		zerolog.SetGlobalLevel(logLevel)

		return delegate.Filter(nodes)
	})

	return framework.SimpleProcessor{Config: &cm, Filter: filter}
}

