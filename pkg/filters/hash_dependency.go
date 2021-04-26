package filters

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/rs/zerolog"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"strings"
)

const hashDependencyAnnotationPrefix string = "kpt.seek.com/hash-dependency"

var podSpecKinds = []string{
	"Deployment",
	"DaemonSet",
	"ReplicaSet",
	"StatefulSet",
}

type HashDependencyFilter struct {
	// Logger specifies the logger to be used by the filter.
	Logger zerolog.Logger
}

func (dh *HashDependencyFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	var output []*yaml.RNode

	for _, node := range input {
		meta, err := node.GetMeta()
		if err != nil {
			return nil, err
		}

		for k, v := range meta.Annotations {
			if strings.HasPrefix(k, hashDependencyAnnotationPrefix) {
				node, err = hashDependency(node, input, meta.Namespace, v)
				if err != nil {
					return nil, err
				}
			}
		}

		podSpecSelector := &framework.Selector{
			Kinds: podSpecKinds,
		}

		hasPodSpec, err := podSpecSelector.Filter([]*yaml.RNode{node})

		if err != nil {
			return nil, err
		}

		if len(hasPodSpec) != 0 {
			podTemplate, err := node.Pipe(yaml.Lookup("spec", "template"))

			if err != nil {
				return nil, fmt.Errorf("failed to find spec.template field, possibly malformed spec: %s", err)
			}

			podAnnotations, err := podTemplate.Pipe(yaml.Lookup("metadata", "annotations"))

			if err != nil {
				return nil, fmt.Errorf("failed to find metadata.annotations from pod template: %s", err)
			}

			_ = podAnnotations.VisitFields(func(annotations *yaml.MapNode) error {
				key := yaml.GetValue(annotations.Key)
				value := yaml.GetValue(annotations.Value)
				if strings.HasPrefix(key, hashDependencyAnnotationPrefix) {
					newPodTemplate, err := hashDependency(podTemplate, input, meta.Namespace, value)
					if err != nil {
						return err
					}
					_, err = node.Pipe(
						yaml.Lookup("spec"),
						yaml.SetField("template", newPodTemplate.Copy()),
					)
					if err != nil {
						return err
					}
				}
				return nil
			})
		}

		output = append(output, node)
	}

	return output, nil
}

func hashDependency(rn *yaml.RNode, nodes []*yaml.RNode, namespace string, hashTarget string) (*yaml.RNode, error) {
	hashTargetTokens := strings.Split(hashTarget, "/")
	if len(hashTargetTokens) != 2 {
		return nil, fmt.Errorf("failed to parse hash target. Expected <kind>/<name>, got %s", hashTarget)
	}
	hashTargetKind := hashTargetTokens[0]
	hashTargetName := hashTargetTokens[1]

	matchingResourceSelector := framework.Selector{
		Names:      []string{hashTargetName},
		Namespaces: []string{namespace},
		Kinds:      []string{hashTargetKind},
	}

	matchingResources, err := matchingResourceSelector.Filter(nodes)

	if err != nil {
		return nil, err
	}

	if len(matchingResources) != 1 {
		return nil, fmt.Errorf("wrong number of matches for hash selector. Expected 1, got %d", len(matchingResources))
	}

	targetBytes := []byte(matchingResources[0].MustString())
	hash := sha256.New()
	_, err = hash.Write(targetBytes)
	if err != nil {
		return nil, err
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	err = rn.PipeE(yaml.SetAnnotation(hashTarget, sum))

	if err != nil {
		return nil, err
	}

	return rn, nil
}
