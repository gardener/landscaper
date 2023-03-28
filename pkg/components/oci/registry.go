package oci

import (
	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type OCIRegistry struct {
	componentResolver ctf.ComponentResolver
}

var _ model.Registry = &OCIRegistry{}

func NewOCIRegistry(componentResolver ctf.ComponentResolver) (model.Registry, error) {
	return &OCIRegistry{
		componentResolver: componentResolver,
	}, nil
}

func (r *OCIRegistry) GetComponentVersion(cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	return nil, nil
}

// temporary
func (r *OCIRegistry) GetComponentResolver() ctf.ComponentResolver {
	return r.componentResolver
}
