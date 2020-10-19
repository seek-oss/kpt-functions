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
  process := func(rn *yaml.RNode) error {
    for _, r := range tr.Config.Spec.Replacements {
      if rn.YNode().Kind == yaml.ScalarNode {
        rn.YNode().Value = strings.ReplaceAll(rn.YNode().Value, r.Token, r.Value)
      } else {
        return tr.replaceTokens(rn)
      }
    }

    return nil
  }

  if rn.YNode().Kind == yaml.MappingNode {
    return rn.VisitFields(func(rn *yaml.MapNode) error {
      return process(rn.Value)
    })
  } else if rn.YNode().Kind == yaml.SequenceNode {
    return rn.VisitElements(func(rn *yaml.RNode) error {
      return process(rn)
    })
  }

  return nil
}
