package main

import (
  "sigs.k8s.io/kustomize/kyaml/fn/framework/frameworktestutil"
  "testing"
)

func TestProcessor(t *testing.T) {
  checker := frameworktestutil.ProcessorResultsChecker{
    TestDataDirectory:        "",
    InputFilename:            "",
    ExpectedOutputFilename:   "",
    ExpectedErrorFilename:    "",
    Processor:                newProcessor,
    UpdateExpectedFromActual: false,
  }

  checker.Assert(t)
}
