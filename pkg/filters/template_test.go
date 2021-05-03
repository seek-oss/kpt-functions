package filters

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework/frameworktestutil"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func TestTemplateRenderer(t *testing.T) {
	newProcessor := func() framework.ResourceListProcessor {
		return framework.SimpleProcessor{
			Filter: &TemplateFilter{},
		}

	}
	checker := frameworktestutil.ProcessorResultsChecker{
		Processor:         newProcessor,
		TestDataDirectory: "template_testdata",
	}

	checker.Assert(t)
}
