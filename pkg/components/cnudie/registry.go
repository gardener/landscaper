package cnudie

import (
	"context"
	"fmt"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type Registry struct {
	componentResolver ctf.ComponentResolver
}

var _ model.Registry = &Registry{}

func NewRegistry(componentResolver ctf.ComponentResolver) model.Registry {
	return &Registry{
		componentResolver: componentResolver,
	}
}

func (r *Registry) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	cd, blobResolver, err := r.componentResolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}

	return newComponentVersion(r, cd, blobResolver), nil
}

// temporary
func (r *Registry) GetComponentResolver() ctf.ComponentResolver {
	return r.componentResolver
}
