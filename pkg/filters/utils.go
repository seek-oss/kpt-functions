package filters

import (
  "sigs.k8s.io/kustomize/kyaml/fn/framework"
  "sigs.k8s.io/kustomize/kyaml/kio"
)

func newProcessor(renderer kio.Filter) framework.ResourceListProcessor {
	return framework.SimpleProcessor{
		Filter: renderer,
	}
}
