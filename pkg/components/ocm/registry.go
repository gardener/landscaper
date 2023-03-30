package ocm

import (
	"context"
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
)

type RegistryAccess struct {
	octx ocm.Context
}

var _ model.RegistryAccess = &RegistryAccess{}

func NewRegistry(octx ocm.Context) model.RegistryAccess {
	return &RegistryAccess{
		octx: octx,
	}
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	var cnudieRepoSpec v2.OCIRegistryRepository
	if err := cdRef.RepositoryContext.DecodeInto(&cnudieRepoSpec); err != nil {
		return nil, err
	}

	var ocmRepoSpec ocm.RepositorySpec
	ocmRepoSpec = ocireg.NewRepositorySpec(cnudieRepoSpec.BaseURL,
		&genericocireg.ComponentRepositoryMeta{ComponentNameMapping: genericocireg.ComponentNameMapping(string(cnudieRepoSpec.ComponentNameMapping))})
	repo, err := r.octx.RepositoryForSpec(ocmRepoSpec)
	if err != nil {
		return nil, err
	}
	defer repo.Close()

	compvers, err := repo.LookupComponentVersion(cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, err
	}

	//defer compvers.Close()

	//cd, blobResolver, err := r.componentResolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	//if err != nil {
	//	return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	//}
	//
	//return newComponentVersion(r, cd, blobResolver), nil
}

// temporary
func (r *RegistryAccess) GetComponentResolver() ctf.ComponentResolver {
	return r.componentResolver
}
