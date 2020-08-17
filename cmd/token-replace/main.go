package main

import (
	"fmt"
	"github.com/seek-oss/kpt-functions/pkg/fns"
	"os"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func main() {
	config := fns.TokenReplaceConfig{}
	tokenReplacer := fns.TokenReplacer{Config: &config}
	resourceList := &framework.ResourceList{FunctionConfig: &config}

	cmd := framework.Command(resourceList, func() error {
		for i := range resourceList.Items {
			if err := resourceList.Items[i].PipeE(&tokenReplacer); err != nil {
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
