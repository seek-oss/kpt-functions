package filters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/rs/zerolog"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// ClusterPackagesKind defines the kind used by the ClusterPackages resource.
	ClusterPackagesKind = "ClusterPackages"
	// ClusterPackagesGroup defines the group used by the ClusterPackages resource.
	ClusterPackagesGroup = "kpt.seek.com"
	// ClusterPackagesVersion defines the version used by the ClusterPackages resource.
	ClusterPackagesVersion = "v1alpha1"
	// ClusterPackagesAPIVersion defines the aggregate group/version used by the ClusterPackages resource.
	ClusterPackagesAPIVersion = ClusterPackagesGroup + "/" + ClusterPackagesVersion

	// SetByClusterOverride defines the set-by value used when Kpt packages setters are set by cluster-level variables.
	SetByClusterOverride = "cluster-override"
	// SetByPackageOverride defines the set-by value used when Kpt packages setters are set by package-level variables.
	SetByPackageOverride = "package-override"
)

// ClusterPackages defines the "client-side CRD" that is managed by the ClusterPackagesFilter. When
// the ClusterPackagesFilter sees a resource that matches this type, it transforms it into a stream
// of resources that consists of all of the resources contained in the referenced Kpt packages.
type ClusterPackages struct {
	// Standard Kubernetes metadata.
	yaml.ResourceMeta `json:",inline" yaml:",inline"`
	// Spec provides the resource specification.
	Spec ClusterPackagesSpec `yaml:"spec,omitempty"`
}

// ClusterPackagesSpec defines the main body of the ClusterPackages resource.
type ClusterPackagesSpec struct {
	// BaseDir specifies the base directory that packages should be written to.
	BaseDir string `yaml:"baseDir,omitempty"`
	// Variables specifies the list of cluster-level variable definitions. Kpt packages referenced in the
	// Packages list may define setters with these names and have their values overridden when they are fetched.
	Variables []Variable `yaml:"variables,omitempty"`
	// Packages specifies the list of Kpt packages that are installed by this cluster.
	Packages []Package `yaml:"packages,omitempty"`
}

// Package defines a Kpt package dependency.
type Package struct {
	// Name specifies the name of the package. This name will be combined with the ClusterPackagesSpec.BaseDir
	// to form the directory that this package should be written to.
	Name string `yaml:"name,omitempty"`
	// Git specifies the upstream Git reference information for the package.
	Git kptfile.Git `yaml:"git,omitempty"`
	// Variables specifies the list of package-level variable definitions. In the case that a package has a setter
	// whose value is specified by both cluster-level and package-level variables, the package-level value will be used.
	Variables []Variable `yaml:"variables,omitempty"`
}

// Variable defines the value for a Kpt package setter.
type Variable struct {
	// Name defines the setter key.
	Name string `yaml:"name,omitempty"`
	// Value defines the setter value.
	Value string `yaml:"value,omitempty"`
}

// ClusterPackagesFilter defines a kio.Filter that processes ClusterPackages custom resources.
type ClusterPackagesFilter struct {
	// CacheDir specifies a directory that is used by the filter to cache Git repositories.
	CacheDir string
	// Logger specifies the logger to be used by the filter.
	Logger zerolog.Logger
}

// Filter implements kio.Filter.Filter.
func (f *ClusterPackagesFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	ctx := context.Background()
	var output []*yaml.RNode
	for _, node := range input {
		meta, err := node.GetMeta()
		if err != nil {
			return nil, err
		}

		// If the current resource isn't a ClusterPackages resource then forward it through.
		if meta.APIVersion != ClusterPackagesAPIVersion || meta.Kind != ClusterPackagesKind {
			output = append(output, node)
			continue
		}

		// The current resource is a ClusterPackages resource so unmarshal it.
		res := &ClusterPackages{}
		if err := yaml.Unmarshal([]byte(node.MustString()), res); err != nil {
			return nil, errors.WrapPrefixf(err, "could not unmarshal input")
		}

		// Fetch and process all of the resources for all of the packages defined in the ClusterPackages spec.
		newNodes, err := f.fetchClusterResources(ctx, res)
		if err != nil {
			return nil, err
		}

		// Append the new package nodes. The ClusterPackages resource is discarded as it has now been fully processed.
		output = append(output, newNodes...)
	}

	return output, nil
}

// fetchClusterResources
func (f *ClusterPackagesFilter) fetchClusterResources(ctx context.Context, res *ClusterPackages) ([]*yaml.RNode, error) {
	var output []*yaml.RNode
	for _, pkg := range res.Spec.Packages {
		nodes, err := f.fetchPackage(ctx, &pkg)
		if err != nil {
			return nil, err
		}

		var setFilters []kio.Filter
		for _, v := range res.Spec.Variables {
			setFilters = append(setFilters, &SetPackageFilter{
				Name:       v.Name,
				Value:      v.Value,
				ListValues: nil,
				SetBy:      SetByClusterOverride,
			})
		}

		for _, v := range pkg.Variables {
			setFilters = append(setFilters, &SetPackageFilter{
				Name:       v.Name,
				Value:      v.Value,
				ListValues: nil,
				SetBy:      SetByPackageOverride,
			})
		}

		for _, f := range setFilters {
			nodes, err = f.Filter(nodes)
			if err != nil {
				return nil, err
			}
		}

		output = append(output, nodes...)
	}

	return output, nil
}

func (f *ClusterPackagesFilter) fetchPackage(ctx context.Context, pkg *Package) ([]*yaml.RNode, error) {
	// The repository for the specified package will be cached at ${cacheDir}/${checksum} where
	// checksum is the sha256 sum of the repository URI.
	checksum := sha256.Sum256([]byte(pkg.Git.Repo))
	repoDir := filepath.Join(f.CacheDir, hex.EncodeToString(checksum[:]))

	// Determine whether the repository has already been cloned and cached.
	isCached := false
	stat, err := os.Stat(repoDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.WrapPrefixf(err, "error checking for directory %s", repoDir)
		}
	} else {
		if stat.IsDir() {
			isCached = true
		} else {
			return nil, errors.Errorf("unexpected non-directory %s exists", repoDir)
		}
	}

	var repo *git.Repository
	if !isCached {
		f.Logger.Debug().Msgf("Cloning repository %s to %s", pkg.Git.Repo, repoDir)

		auth, err := ssh.NewPublicKeysFromFile("git", "/Users/aeldridge/.ssh/id_rsa", "")
		if err != nil {
			return nil, errors.WrapPrefixf(err, "error retrieving Git private key information")
		}

		repo, err = git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
			URL:  pkg.Git.Repo,
			Auth: auth,
		})
		if err != nil {
			return nil, errors.WrapPrefixf(err, "error cloning Git repository %s", pkg.Git.Repo)
		}
	} else {
		f.Logger.Debug().Msgf("Using %s in %s", pkg.Git.Repo, f.CacheDir)

		repo, err = git.PlainOpen(repoDir)
		if err != nil {
			return nil, errors.WrapPrefixf(err, "error opening Git repository %s", pkg.Git.Repo)
		}
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, errors.WrapPrefixf(err, "error obtaining worktree for repository %s", pkg.Git.Repo)
	}

	if err := w.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(pkg.Git.Ref)}); err != nil {
		return nil, errors.WrapPrefixf(err, "error checking out ref %s for repository %s", pkg.Git.Ref, pkg.Git.Repo)
	}

	reader := kio.LocalPackageReader{
		PackagePath:    filepath.Join(repoDir, pkg.Git.Directory),
		MatchFilesGlob: append(kio.DefaultMatch, kptfile.KptFileName),
	}

	nodes, err := reader.Read()
	if err != nil {
		return nil, errors.WrapPrefixf(err, "error reading resources from %s", repoDir)
	}

	return nodes, nil
}
