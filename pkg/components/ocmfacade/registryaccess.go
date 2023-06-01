package ocmfacade

import (
	"context"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type RegistryAccess struct {
	octx ocm.Context
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (rcompvers model.ComponentVersion, rerr error) {
	octx := ocm.DefaultContext()
	ocmrepo, err := octx.RepositoryForConfig(cdRef.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
	if err != nil {
		return nil, err
	}
	defer errors.PropagateError(&rerr, ocmrepo.Close)

	compvers, err := ocmrepo.LookupComponentVersion(cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, err
	}

	return &ComponentVersion{
		componentVersionAccess: compvers,
	}, err
}
