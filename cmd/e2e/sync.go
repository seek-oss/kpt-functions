package main

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/fieldmeta"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/cmd/config/ext"

	"sigs.k8s.io/kustomize/cmd/config/configcobra"

	"golang.org/x/sync/errgroup"

	"github.com/seek-oss/kpt-functions/internal/get"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	setByClusterOverride = "cluster-override"
	setByPackageOverride = "package-override"
)

func init() {
	// Since we're using Kustomize packages to run setters against the Kpt packages,
	// we need to tell Kustomize that we're Kpt's filename convention rather than Kustomize's.
	ext.KRMFileName = func() string {
		return kptfile.KptFileName
	}
}

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

	kpath := filepath.Join(pkgDir, ext.KRMFileName())
	knode, err := yaml.ReadFile(kpath)
	if err != nil {
		return errors.WrapPrefixf(err, "could not read Kptfile %s", kpath)
	}

	hasSetter := func(name string) (bool, error) {
		key := fieldmeta.SetterDefinitionPrefix + name
		n, err := knode.Pipe(yaml.Lookup("openAPI", "definitions", key))
		return n != nil, err
	}

	setVar := func(v Variable, setBy string) error {
		setCmd := configcobra.Set("")
		setCmd.SetOut(io.Discard)
		setCmd.SetErr(io.Discard)
		setCmd.SetArgs([]string{pkgDir, v.Name, v.Value, "--set-by", setBy})
		if err := setCmd.Execute(); err != nil {
			return errors.WrapPrefixf(err, "error setting variable %s on package %s", v.Name, pkgDir)
		}
		return nil
	}

	// Apply cluster-level variables to the package
	for _, v := range clusterConfig.Spec.Variables {
		ok, err := hasSetter(v.Name)
		if err != nil {
			return err
		}

		if !ok {
			// Setter doesn't exist for cluster-level variable for this package
			continue
		}

		logger.Debug().Msgf("Dependency %s: setting cluster-level variable %s", pkgDir, v.Name)
		if err := setVar(v, setByClusterOverride); err != nil {
			return err
		}
	}

	// Apply package-level variables to the package
	for _, v := range dep.Variables {
		ok, err := hasSetter(v.Name)
		if err != nil {
			return err
		}

		if !ok {
			return errors.Errorf("dependency %s specifies variable %s but no setter exists", pkgDir, v.Name)
		}

		logger.Debug().Msgf("Dependency %s: setting package-level variable %s", pkgDir, v.Name)
		if err := setVar(v, setByPackageOverride); err != nil {
			return err
		}
	}

	logger.Debug().Msgf("Dependency %s has been updated to ref %s", pkgDir, dep.Git.Ref)
	return nil
}
