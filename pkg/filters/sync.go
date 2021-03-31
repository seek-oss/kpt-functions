package filters

//import (
//	"context"
//	"crypto/sha256"
//	"encoding/hex"
//	"io/ioutil"
//	"os"
//	"path/filepath"
//
//	"sigs.k8s.io/kustomize/kyaml/kio"
//	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
//
//	"github.com/go-git/go-git/v5"
//	"github.com/go-git/go-git/v5/plumbing"
//	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
//
//	"github.com/rs/zerolog"
//
//	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
//
//	"sigs.k8s.io/kustomize/cmd/config/ext"
//	"sigs.k8s.io/kustomize/kyaml/errors"
//	"sigs.k8s.io/kustomize/kyaml/yaml"
//)
//
//const (
//	SetByClusterOverride = "cluster-override"
//	SetByPackageOverride = "package-override"
//)
//
//func init() {
//	// Since we're using Kustomize packages to run setters against the Kpt packages,
//	// we need to tell Kustomize that we're Kpt's filename convention rather than Kustomize's.
//	ext.KRMFileName = func() string {
//		return kptfile.KptFileName
//	}
//}
//
//type ClusterPackagesFilter struct {
//	ClusterPackagesFilterConfig *ClusterPackagesFilterConfig
//	Logger zerolog.Logger
//}
//
//type ClusterPackagesFilterConfig struct {
//	Data struct {
//		LogLevel string `yaml:"logLevel,omitempty"`
//	} `yaml:"data,omitempty"`
//}
//
//// Default implements framework.Defaulter.
//func (c *ClusterPackagesFilterConfig) Default() error {
//	if c.Data.LogLevel == "" {
//		c.Data.LogLevel = "info"
//	}
//	return nil
//}
//
//// Validate implements framework.Validator.
//func (c *ClusterPackagesFilterConfig) Validate() error {
//	return nil
//}
//
//func (f *ClusterPackagesFilter) ClusterPackagesFilter(input []*yaml.RNode) ([]*yaml.RNode, error) {
//	ctx := context.Background()
//	var output []*yaml.RNode
//
//	for _, node := range input {
//		meta, err := node.GetMeta()
//		if err != nil {
//			return nil, err
//		}
//
//		if meta.APIVersion != ClusterPackagesAPIVersion || meta.Kind != ClusterPackagesKind {
//			output = append(output, node)
//			continue
//		}
//
//		res := &ClusterPackages{}
//		if err := yaml.Unmarshal([]byte(node.MustString()), res); err != nil {
//			return nil, errors.WrapPrefixf(err, "could not unmarshal input")
//		}
//
//		newNodes, err := f.fetchPackageResources(ctx, res)
//		if err != nil {
//			return nil, err
//		}
//
//		output = append(output, newNodes...)
//	}
//
//	return output, nil
//}
//
//func (f *ClusterPackagesFilter) fetchPackageResources(ctx context.Context, res *ClusterPackages) ([]*yaml.RNode, error) {
//	tmpDir, err := ioutil.TempDir("", "")
//	if err != nil {
//		return nil, errors.WrapPrefixf(err, "")
//	}
//
//	defer os.RemoveAll(tmpDir)
//
//	var output []*yaml.RNode
//	var repo *git.Repository
//	for _, pkg := range res.Spec.Packages {
//		checksum := sha256.Sum256([]byte(pkg.Git.Repo))
//		repoDir := filepath.Join(tmpDir, hex.EncodeToString(checksum[:]))
//		stat, err := os.Stat(repoDir)
//		cloned := false
//		if err != nil {
//			if !os.IsNotExist(err) {
//				return nil, errors.WrapPrefixf(err, "error checking for directory %s", repoDir)
//			}
//		} else {
//			if stat.IsDir() {
//				cloned = true
//			} else {
//				return nil, errors.Errorf("unexpected non-directory %s exists", repoDir)
//			}
//		}
//
//		if !cloned {
//			f.Logger.Debug().Msgf("Cloning %s to %s", pkg.Git.Repo, repoDir)
//
//			auth, err := ssh.NewPublicKeysFromFile("git", "/Users/aeldridge/.ssh/id_rsa", "")
//			if err != nil {
//				return nil, errors.WrapPrefixf(err, "error retrieving Git private key information")
//			}
//
//			repo, err = git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
//				URL:  pkg.Git.Repo,
//				Auth: auth,
//			})
//			if err != nil {
//				return nil, errors.WrapPrefixf(err, "error cloning Git repository %s", pkg.Git.Repo)
//			}
//		} else {
//			f.Logger.Debug().Msgf("Using %s in %s", pkg.Git.Repo, tmpDir)
//
//			repo, err = git.PlainOpen(repoDir)
//			if err != nil {
//				return nil, errors.WrapPrefixf(err, "error opening Git repository %s", pkg.Git.Repo)
//			}
//		}
//
//		w, err := repo.Worktree()
//		if err != nil {
//			return nil, errors.WrapPrefixf(err, "error obtaining worktree for repository %s", pkg.Git.Repo)
//		}
//
//		if err := w.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(pkg.Git.Ref)}); err != nil {
//			return nil, errors.WrapPrefixf(err, "error checking out ref %s for repository %s", pkg.Git.Ref, pkg.Git.Repo)
//		}
//
//		reader := kio.LocalPackageReader{
//			PackagePath:    filepath.Join(repoDir, pkg.Git.Directory),
//			MatchFilesGlob: append(kio.DefaultMatch, kptfile.KptFileName),
//		}
//
//		pkgNodes, err := reader.Read()
//		if err != nil {
//			return nil, errors.WrapPrefixf(err, "error reading resources from %s", repoDir)
//		}
//
//		for _, n := range pkgNodes {
//			if err := n.PipeE(UpdateAnnotation(kioutil.PathAnnotation, func(s string) (string, error) {
//				return filepath.Join(pkg.Name, s), nil
//			})); err != nil {
//				return nil, err
//			}
//		}
//
//		output = append(output, pkgNodes...)
//	}
//
//	return output, nil
//}

//
//func (f *ClusterPackagesFilter) fetchPackageResources(ctx context.Context, clusterConfig *ClusterPackages) error {
//	group, _ := errgroup.WithContext(ctx)
//	for _, dep := range clusterConfig.Spec.Packages {
//		group.Go(func() error {
//			return f.sync(ctx, clusterConfig, &dep)
//		})
//	}
//
//	return errors.WrapPrefixf(group.Wait(), "error waiting for syncing of packages")
//}
//
//func (f *ClusterPackagesFilter) sync(ctx context.Context, res *ClusterPackages) ([]*yaml.RNode, error) {
//	pkgDir := filepath.Join(f.ClusterPackagesFilterConfig.Data.BaseDir, dep.Name)
//	logger.Debug().Msgf("Refreshing dependency %s", pkgDir)
//
//	getCmd := get.Command{
//		Git:         dep.Git,
//		Destination: pkgDir,
//		Name:        pkgDir,
//		Clean:       true,
//	}
//	if err := getCmd.Run(); err != nil {
//		return errors.WrapPrefixf(err, "error performing kpt get operation on dependency %s", pkgDir)
//	}
//
//	kpath := filepath.Join(pkgDir, kptfile.KptFileName)
//	knode, err := yaml.ReadFile(kpath)
//	if err != nil {
//		return errors.WrapPrefixf(err, "could not read Kptfile %s", kpath)
//	}
//
//	hasSetter := func(name string) (bool, error) {
//		key := fieldmeta.SetterDefinitionPrefix + name
//		n, err := knode.Pipe(yaml.Lookup("openAPI", "definitions", key))
//		return n != nil, err
//	}
//
//	setVar := func(v Variable, setBy string) error {
//		setCmd := configcobra.Set("")
//		setCmd.SetOut(io.Discard)
//		setCmd.SetErr(io.Discard)
//		setCmd.SetArgs([]string{pkgDir, v.Name, v.Value, "--set-by", setBy})
//		if err := setCmd.Execute(); err != nil {
//			return errors.WrapPrefixf(err, "error setting variable %s on package %s", v.Name, pkgDir)
//		}
//		return nil
//	}
//
//	// Apply cluster-level variables to the package
//	for _, v := range clusterConfig.Spec.Variables {
//		ok, err := hasSetter(v.Name)
//		if err != nil {
//			return err
//		}
//
//		if !ok {
//			// Setter doesn't exist for cluster-level variable for this package
//			continue
//		}
//
//		logger.Debug().Msgf("Packages %s: setting cluster-level variable %s", pkgDir, v.Name)
//		if err := setVar(v, SetByClusterOverride); err != nil {
//			return err
//		}
//	}
//
//	// Apply package-level variables to the package
//	for _, v := range dep.Variables {
//		ok, err := hasSetter(v.Name)
//		if err != nil {
//			return err
//		}
//
//		if !ok {
//			return errors.Errorf("dependency %s specifies variable %s but no setter exists", pkgDir, v.Name)
//		}
//
//		logger.Debug().Msgf("Packages %s: setting package-level variable %s", pkgDir, v.Name)
//		if err := setVar(v, SetByPackageOverride); err != nil {
//			return err
//		}
//	}
//
//	logger.Debug().Msgf("Packages %s has been updated to ref %s", pkgDir, dep.Git.Ref)
//	return nil
//}
