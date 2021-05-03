package filters

import (
  "sigs.k8s.io/kustomize/kyaml/fn/framework"
  "sigs.k8s.io/kustomize/kyaml/fn/framework/frameworktestutil"
  "testing"
)

func TestSet(t *testing.T) {
  var tests = []struct{
    name string
    filter *SetPackageFilter
    testdataPath string
  }{
    {
      name: "set-single-value",
      filter:       &SetPackageFilter{
        Name:       "replicas",
        Value:      "7",
        ListValues: nil,
        SetBy:      SetByClusterOverride,
      },
      testdataPath:  "set_testdata/value",
    },
    {
      name: "set-list-values-multiple-values",
      filter:       &SetPackageFilter{
        Name:       "hosts",
        Value:      "",
        ListValues: []string{"test.com", "example-2.com", "hello.com"},
        SetBy:      SetByClusterOverride,
      },
      testdataPath:  "set_testdata/list_values/multiple",
    },
    {
      name: "set-list-values-single-value",
      filter:       &SetPackageFilter{
        Name:       "hosts",
        Value:      "",
        ListValues: []string{"test.com"},
        SetBy:      SetByClusterOverride,
      },
      testdataPath:  "set_testdata/list_values/single",
    },
  }

  for _, test := range tests {
    t.Run(test.name, func(t *testing.T) {
      newProcessor := func() framework.ResourceListProcessor {
        return framework.SimpleProcessor{
          Filter: test.filter,
        }
      }

      checker := frameworktestutil.ProcessorResultsChecker{
        Processor:         newProcessor,
        TestDataDirectory: test.testdataPath,
      }

      checker.Assert(t)
    })
  }
}
