package cnudie

import (
	"context"
	"fmt"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type RegistryAccess struct {
	componentResolver ctf.ComponentResolver
}

var _ model.RegistryAccess = &RegistryAccess{}

func NewRegistry(componentResolver ctf.ComponentResolver) model.RegistryAccess {
	return &RegistryAccess{
		componentResolver: componentResolver,
	}
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	cd, blobResolver, err := r.componentResolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}

	return newComponentVersion(r, cd, blobResolver), nil
}

// temporary
func (r *RegistryAccess) GetComponentResolver() ctf.ComponentResolver {
	return r.componentResolver
}
