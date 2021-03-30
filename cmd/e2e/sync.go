package main

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/cmd/config/configcobra"

	"golang.org/x/sync/errgroup"

	"github.com/seek-oss/kpt-functions/internal/get"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewSyncProcessor() framework.ResourceListProcessor {
	config := &SyncConfig{}
	filter := &SyncFilter{Config: config}
	return framework.SimpleProcessor{
		Config: config,
		Filter: filter,
	}
}

type SyncFilter struct {
	Config *SyncConfig
}

type SyncConfig struct {
	Data struct {
		LogLevel string `yaml:"logLevel,omitempty"`
		BaseDir  string `yaml:"baseDir,omitempty"`
	} `yaml:"data,omitempty"`
}

// Default implements framework.Defaulter.
func (c *SyncConfig) Default() error {
	if c.Data.LogLevel == "" {
		c.Data.LogLevel = "info"
	}
	return nil
}

// Validate implements framework.Validator.
func (c *SyncConfig) Validate() error {
	if c.Data.BaseDir == "" {
		return errors.Errorf("no base directory specified")
	}

	stat, err := os.Stat(c.Data.BaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf("base directory %s does not exist", c.Data.BaseDir)
		}
		return errors.WrapPrefixf(err, "error reading base directory %s", c.Data.BaseDir)
	}

	if !stat.IsDir() {
		return errors.Errorf("%s is not a directory")
	}

	return nil
}

func (f *SyncFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	if len(nodes) != 1 {
		return nil, errors.Errorf("expected single input nodes but got %d", len(nodes))
	}

	clusterConfig := &ClusterConfig{}
	if err := yaml.Unmarshal([]byte(nodes[0].MustString()), clusterConfig); err != nil {
		return nil, errors.WrapPrefixf(err, "could not unmarshal input")
	}

	ctx := context.Background()

	if err := f.cleanBaseDir(ctx); err != nil {
		return nil, err
	}

	if err := f.syncAll(ctx, clusterConfig); err != nil {
		return nil, err
	}

	return nil, nil
}

func (f *SyncFilter) cleanBaseDir(ctx context.Context) error {
	files, err := ioutil.ReadDir(f.Config.Data.BaseDir)
	if err != nil {
		return errors.WrapPrefixf(err, "could not read base directory %s", f.Config.Data.BaseDir)
	}

	group, _ := errgroup.WithContext(ctx)

	for _, file := range files {
		if file.IsDir() {
			group.Go(func() error {
				dir := filepath.Join(f.Config.Data.BaseDir, file.Name())
				logger.Debug().Msgf("Removing existing package directory %s", dir)
				if err := os.RemoveAll(dir); err != nil {
					return errors.WrapPrefixf(err, "could not delete existing package directory %s", dir)
				}
				return nil
			})
		}
	}

	return errors.WrapPrefixf(group.Wait(), "error waiting for deletion of existing packages")
}

func (f *SyncFilter) syncAll(ctx context.Context, clusterConfig *ClusterConfig) error {
	group, _ := errgroup.WithContext(ctx)
	for _, dep := range clusterConfig.Spec.Dependencies {
		group.Go(func() error {
			return f.sync(ctx, clusterConfig, &dep)
		})
	}

	return errors.WrapPrefixf(group.Wait(), "error waiting for syncing of packages")
}

func (f *SyncFilter) sync(ctx context.Context, clusterConfig *ClusterConfig, dep *Dependency) error {
	pkgDir := filepath.Join(f.Config.Data.BaseDir, dep.Name)
	logger.Debug().Msgf("Refreshing dependency %s", pkgDir)

	getCmd := get.Command{
		Git:         dep.Git,
		Destination: pkgDir,
		Name:        pkgDir,
		Clean:       true,
	}
	if err := getCmd.Run(); err != nil {
		return errors.WrapPrefixf(err, "error performing kpt get operation on dependency %s", pkgDir)
	}

	setVar := func(v Variable) error {
		setCmd := configcobra.Set("")
		setCmd.SetOut(io.Discard)
		setCmd.SetErr(io.Discard)
		setCmd.SetArgs([]string{filepath.Join(pkgDir, v.Name, v.Value, "--set-by", "cluster-override")})
		if err := setCmd.Execute(); err != nil {
			return errors.WrapPrefixf(err, "error setting variable %s on package %s", v.Name, pkgDir)
		}
		return nil
	}

	for _, v := range clusterConfig.Spec.Variables {
		logger.Debug().Msgf("Dependency %s: setting cluster-level variable %s", pkgDir, v.Name)
		if err := setVar(v); err != nil {
			return err
		}
	}

	for _, v := range dep.Variables {
		logger.Debug().Msgf("Dependency %s: setting package-level variable %s", pkgDir, v.Name)
		if err := setVar(v); err != nil {
			return err
		}
	}

	logger.Debug().Msgf("Dependency %s has been updated to ref %s", pkgDir, dep.Git.Ref)
	return nil
}
