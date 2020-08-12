package fns

import (
  "sigs.k8s.io/kustomize/kyaml/yaml"
  "strings"
)

const (
  // Label used to enable token replacement
  tokenReplaceEnabledLabelKey = "kpt.seek.com/token-replace"

  // Label value used to enable token replacement
  tokenReplaceEnabledLabelValue = "enabled"
)

type TokenReplaceConfig struct {
  Spec Spec `yaml:"spec,omitempty"`
}

type Spec struct {
  Replacements []Replacement `yaml:"replacements,omitempty"`
}

type Replacement struct {
  Token string `yaml:"token,omitempty"`
  Value string `yaml:"value,omitempty"`
}


type TokenReplacer struct {
  Config *TokenReplaceConfig
}

func (tr *TokenReplacer) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
  meta, err := rn.GetMeta()
  if err != nil {
    return nil, err
  }

  if meta.Labels[tokenReplaceEnabledLabelKey] != tokenReplaceEnabledLabelValue {
    return rn, nil
  }

  // We only perform token replacement on ConfigMaps for now
  if meta.Kind == "ConfigMap" {
    return rn, tr.processConfigMap(rn)
  }

  return rn, nil
}

func (tr *TokenReplacer) processConfigMap(rn *yaml.RNode) error {
  data := rn.Field("data")
  if data == nil {
    return nil
  }

  return data.Value.VisitFields(func(node *yaml.MapNode) error {
    for _, r := range tr.Config.Spec.Replacements {
      yn := node.Value.YNode()
      if yn.Kind == yaml.ScalarNode {
        yn.Value = strings.ReplaceAll(yn.Value, r.Token, r.Value)
      }
    }
    return nil
  })
}

