package filters

import (
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func KptfileFilter() kio.Filter {
	return findAll(isKptfile)
}

func NotKptfileFilter() kio.Filter {
	return findAll(isNotKptfile)
}

// isKptfile returns true if the specified node is a Kptfile, false otherwise.
func isKptfile(node *yaml.RNode) bool {
	meta, err := node.GetMeta()
	if err != nil {
		return false
	}

	return meta.TypeMeta == kptfile.TypeMeta.TypeMeta
}

// isNotKptfile returns true if the specified node is not a Kptfile, false otherwise.
func isNotKptfile(node *yaml.RNode) bool {
	return !isKptfile(node)
}

func hasKptfileSetter(node *yaml.RNode, name string) bool {
	key := fieldmeta.SetterDefinitionPrefix + name
	oa, err := node.Pipe(yaml.Lookup(openapi.SupplementaryOpenAPIFieldName, openapi.Definitions, key))
	if err != nil {
		return false
	}

	return oa != nil
}

func findAll(p func(*yaml.RNode) bool) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		var output []*yaml.RNode
		for _, n := range nodes {
			if p(n) {
				output = append(output, n)
			}
		}

		return output, nil
	})
}
