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

	if meta.Annotations[tokenReplaceEnabledLabelKey] != tokenReplaceEnabledLabelValue {
		return rn, nil
	}

  return rn, tr.replaceTokens(rn)
}

func (tr *TokenReplacer) replaceTokens(rn *yaml.RNode) error {
  switch rn.YNode().Kind {
  case yaml.MappingNode:
    return rn.VisitFields(func(rn *yaml.MapNode) error {
      return tr.replaceTokens(rn.Value)
    })

  case yaml.SequenceNode:
    return rn.VisitElements(func(rn *yaml.RNode) error {
      return tr.replaceTokens(rn)
    })

  case yaml.ScalarNode:
    for _, r := range tr.Config.Spec.Replacements {
      rn.YNode().Value = strings.ReplaceAll(rn.YNode().Value, r.Token, r.Value)
    }
  }

  return nil
}
