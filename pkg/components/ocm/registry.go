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
	// Muss noch ersetzt werden, macht langfristig keinen Sinn, auf Datentypen aus dem Legacy Code aufzubauen
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

	compvers, err := repo.LookupComponentVersion(cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, err
	}

	return newComponentVersion(compvers), nil
}

// temporary
func (r *RegistryAccess) GetComponentResolver() ctf.ComponentResolver {
	panic("to be removed")
}
