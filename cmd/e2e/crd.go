package main

import "github.com/GoogleContainerTools/kpt/pkg/kptfile"

type ClusterConfig struct {
	Spec ClusterSpec
}

type ClusterSpec struct {
	Variables    []Variable   `yaml:"variables,omitempty"`
	Dependencies []Dependency `yaml:"dependencies,omitempty"`
}

type Dependency struct {
	Name      string      `yaml:"name,omitempty"`
	Git       kptfile.Git `yaml:"git,omitempty"`
	Variables []Variable  `yaml:"variables,omitempty"`
}

type Variable struct {
	Name  string `yaml:"name,omitempty"`
	Value string `yaml:"value,omitempty"`
}
