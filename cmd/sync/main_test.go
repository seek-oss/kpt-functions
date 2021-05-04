package main

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework/frameworktestutil"
	"testing"
)

func TestProcessor(t *testing.T) {
	checker := frameworktestutil.ProcessorResultsChecker{
		Processor: newProcessor,
	}

	checker.Assert(t)
}
