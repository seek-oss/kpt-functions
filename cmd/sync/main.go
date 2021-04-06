package main

import (
  "github.com/seek-oss/kpt-functions/pkg/log"
  "io/ioutil"
	"os"
	"strconv"

	"github.com/mitchellh/go-homedir"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

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
	cacheDirFunctionArg     = "cacheDir"
	keepCacheFunctionArg    = "keepCache"
	gitKeySecretFunctionArg = "gitKeySecretID"
	gitKeyFileFunctionArg   = "gitKeyFile"

	defaultLogLevel   = zerolog.InfoLevel
	defaultKeepCache  = false
	defaultGitKeyFile = "~/.ssh/id_rsa"
)

// logger is the configured zerolog Logger instance.
var logger zerolog.Logger

// Entry point for the sync custom Kpt function.
func main() {
  logger = log.GetLogger(defaultLogLevel)
	if err := realMain(); err != nil {
		logger.Fatal().Err(err).Msgf("Error performing sync operation")
	}
}

// realMain executes the sync operation and returns any errors.
func realMain() error {
	proc := newProcessor()
	rw, err := readWriter()
	if err != nil {
		return err
	}

	return framework.Execute(proc, rw)
}

// newProcessor returns the framework.ResourceListProcessor for the custom sync function.
func newProcessor() framework.ResourceListProcessor {
	var cm v1.ConfigMap
	delegate := &filters.ClusterPackagesFilter{Logger: logger}

	filter := kio.FilterFunc(func(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
		var err error
		var ok bool

		if secretID, ok := cm.Data[gitKeySecretFunctionArg]; ok {
			key, err := readGitPrivateKeySecret(secretID)
			if err != nil {
				return nil, err
			}

			delegate.GitPrivateKey = key
		} else {
			f, ok := cm.Data[gitKeyFileFunctionArg]
			if !ok {
				f, err = homedir.Expand(defaultGitKeyFile)
				if err != nil {
					return nil, err
				}
				logger.Info().Msgf("No Git key specified - falling back to %s", f)
			}

			key, err := readGitPrivateKeyFile(f)
			if err != nil {
				return nil, err
			}

			delegate.GitPrivateKey = key
		}

		logLevel := defaultLogLevel
		if v, ok := cm.Data[logLevelFunctionArg]; ok {
			logLevel, err = zerolog.ParseLevel(v)
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not parse log level")
			}
		}

		zerolog.SetGlobalLevel(logLevel)

		keepCache := defaultKeepCache

		delegate.CacheDir, ok = cm.Data[cacheDirFunctionArg]
		if ok {
			keepCache = true
		} else {
			delegate.CacheDir, err = ioutil.TempDir("", "")
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not create temporary cache directory")
			}
		}

		if v, ok := cm.Data[keepCacheFunctionArg]; ok {
			keepCache, err = strconv.ParseBool(v)
			if err != nil {
				return nil, errors.WrapPrefixf(err, "could not parse keepCache argument")
			}
		}

		defer func() {
			if keepCache {
				return
			}

			if err := os.RemoveAll(delegate.CacheDir); err != nil {
				logger.Fatal().Err(err).Msgf("Could not delete cache directory %s", delegate.CacheDir)
			}
		}()

		return delegate.Filter(nodes)
	})

	return framework.SimpleProcessor{Config: &cm, Filter: filter}
}

// readWriter returns a kio.ByteReadWriter that is configured to read from stdin if no command line argument
// has been specified. If command line arguments are specified, the first argument is assumed to be a file
// containing a framework.ResourceList - this can be useful for debugging locally.
func readWriter() (*kio.ByteReadWriter, error) {
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

// readGitPrivateKeySecret reads the Git private key file from AWS Secrets Manager.
func readGitPrivateKeySecret(secretID string) ([]byte, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	client := secretsmanager.New(sess)
	req := secretsmanager.GetSecretValueInput{SecretId: &secretID}
	res, err := client.GetSecretValue(&req)
	if err != nil {
		return nil, err
	}

	return []byte(*res.SecretString), nil
}

// readGitPrivateKeyFile reads the Git private key file from the filesystem.
func readGitPrivateKeyFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}
