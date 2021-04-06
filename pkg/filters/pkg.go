package filters

import (
	kpt "github.com/GoogleContainerTools/kpt/commands"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/cmd/config/ext"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
)

func init() {
	// Since we're using the Kustomize library to run setters against the Kpt packages, we
	// need to tell Kustomize that we're using Kpt's filename convention rather than Kustomize's.
	ext.KRMFileName = func() string {
		return kptfile.KptFileName
	}

	// Use Kpt's $kpt-set shorthand rather than OpenAPI's.
	fieldmeta.SetShortHandRef(kpt.ShortHandRef)
}
