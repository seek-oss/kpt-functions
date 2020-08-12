package fns

import (
  "crypto/sha256"
  "encoding/hex"
  "fmt"
  "sigs.k8s.io/kustomize/kyaml/yaml"
  "strings"
)

const hashDependencyAnnotationPrefix = "kpt.seek.com/hash-dependency"

type HashDependencyConfig struct {
  Spec Spec `yaml:"spec,omitempty"`
}

type DependencyHasher struct {
  ResourceListItems []*yaml.RNode
}

func (dh *DependencyHasher) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
  meta, err := rn.GetMeta()
  if err != nil {
    return nil, err
  }

  for k, v := range meta.Annotations {
    if strings.HasPrefix(k, hashDependencyAnnotationPrefix) {
      err := dh.hashDependency(rn, v)
      if err != nil {
        return rn, err
      }
    }
  }

  return rn, nil
}

func (dh *DependencyHasher) hashDependency(rn *yaml.RNode, hashTarget string) (error) {
  hashTargetTokens := strings.Split(hashTarget, "/")
  if len(hashTargetTokens) != 2 {
    return fmt.Errorf("failed to parse hash target. Expected <kind>/<name>, got %s", hashTarget)
  }
  hashTargetKind := hashTargetTokens[0]
  hashTargetName := hashTargetTokens[1]

  meta, err := rn.GetMeta()
  if err != nil {
    return err
  }

  for i := range dh.ResourceListItems {
    target := dh.ResourceListItems[i]
    targetMeta, err := target.GetMeta()
    if err != nil {
      return err
    }
    sameNamespace := strings.EqualFold(targetMeta.Namespace, meta.Namespace)
    matchingKind := strings.EqualFold(targetMeta.Kind, hashTargetKind)
    matchingName := strings.EqualFold(targetMeta.Name, hashTargetName)

    if sameNamespace && matchingKind && matchingName {
      targetBytes := []byte(target.MustString())
      hash := sha256.New()
      _, err := hash.Write(targetBytes)
      if err != nil {
        return err
      }
      sum := hex.EncodeToString(hash.Sum(nil))
      return rn.PipeE(yaml.SetAnnotation(hashTarget, sum))
    }
  }
  return nil
}
