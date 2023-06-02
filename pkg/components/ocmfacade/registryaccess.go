package ocmfacade

import (
	"context"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type RegistryAccess struct {
	octx    ocm.Context
	session ocm.Session
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (rcompvers model.ComponentVersion, rerr error) {
	spec, err := r.octx.RepositorySpecForConfig(cdRef.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
	if err != nil {
		return nil, err
	}
	ocmrepo, err := r.session.LookupRepository(r.octx, spec)
	if err != nil {
		return nil, err
	}
	compvers, err := r.session.LookupComponentVersion(ocmrepo, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, err
	}
	return &ComponentVersion{
		componentVersionAccess: compvers,
	}, err
}

func (r *RegistryAccess) Close() error {
	err := r.session.Close()
	if err != nil {
		return err
	}
	return nil
}
