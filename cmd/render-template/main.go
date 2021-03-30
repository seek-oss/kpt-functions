package main

import (
	"fmt"
	"os"

	"github.com/go-errors/errors"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func main() {
	proc := newProcessor()
	if err := framework.Execute(proc, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing resources: %s\n", err)
		if e, ok := err.(*errors.Error); ok {
			trace := e.ErrorStack()
			fmt.Fprintf(os.Stderr, "Stack trace: %s\n", trace)
		}

		os.Exit(1)
	}
}

func newProcessor() framework.ResourceListProcessor {
	config := &RenderTemplateConfig{}
	renderer := &TemplateRenderer{Config: config}
	return framework.SimpleProcessor{
		Config: config,
		Filter: renderer,
	}
}
