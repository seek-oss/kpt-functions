package main

import (
	"fmt"
	"os"

	"github.com/seek-oss/kpt-functions/pkg/fns"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func main() {
	config := &fns.RenderTemplateConfig{}
	renderer := &fns.TemplateRenderer{Config: config}
	resourceList := &framework.ResourceList{FunctionConfig: config}

	cmd := framework.Command(resourceList, func() error {
		for i := range resourceList.Items {
			if err := resourceList.Items[i].PipeE(renderer); err != nil {
				return err
			}
		}
		return nil
	})

	// TODO: Remove
	f, err := os.Open("fake.yaml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cmd.SetIn(f)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing resources: %s\n", err)
		os.Exit(1)
	}
}
