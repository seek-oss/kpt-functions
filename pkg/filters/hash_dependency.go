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

const hashDependencyAnnotationPrefix = "kpt.seek.com/hash-dependency"

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
        node, err = hashDependency(node, input, v)
        if err != nil {
          return nil, err
        }
      }
    }

    output = append(output, node)
  }

  return output, nil
}

func hashDependency(rn *yaml.RNode, nodes []*yaml.RNode, hashTarget string) (*yaml.RNode, error) {
	hashTargetTokens := strings.Split(hashTarget, "/")
	if len(hashTargetTokens) != 2 {
		return nil, fmt.Errorf("failed to parse hash target. Expected <kind>/<name>, got %s", hashTarget)
	}
	hashTargetKind := hashTargetTokens[0]
	hashTargetName := hashTargetTokens[1]

	meta, err := rn.GetMeta()
	if err != nil {
		return nil, err
	}

	matchingResourceSelector := framework.Selector{
	  Names: []string{hashTargetName},
	  Namespaces: []string{meta.Namespace},
	  Kinds: []string{hashTargetKind},
  }

  matchingResources, err := matchingResourceSelector.Filter(nodes)

  if err != nil {
    return nil, err
  }

  if len(matchingResources) != 1 {
    return nil, fmt.Errorf("found multiple targets that matched hash selector. Expected 1, got %d", len(matchingResources))
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
