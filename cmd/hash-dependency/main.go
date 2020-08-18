package main

import (
	"fmt"
	"github.com/seek-oss/kpt-functions/pkg/fns"
	"os"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func main() {
	config := fns.HashDependencyConfig{}
	resourceList := &framework.ResourceList{FunctionConfig: &config}

	cmd := framework.Command(resourceList, func() error {
    dependencyHasher := fns.DependencyHasher{ResourceListItems: resourceList.Items}
		for i := range resourceList.Items {
			if err := resourceList.Items[i].PipeE(&dependencyHasher); err != nil {
				return err
			}
		}
		return nil
	})

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing resources: %s\n", err)
		os.Exit(1)
	}
}
