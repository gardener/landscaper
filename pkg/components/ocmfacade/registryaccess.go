package ocmfacade

import (
	"context"
	"reflect"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	_ "github.com/gardener/landscaper/pkg/components/ocmfacade/repository/inline"
	_ "github.com/gardener/landscaper/pkg/components/ocmfacade/repository/local"
)

type RegistryAccess struct {
	octx             ocm.Context
	session          ocm.Session
	inlineSpec       ocm.RepositorySpec
	inlineRepository ocm.Repository
	resolver         ocm.ComponentVersionResolver
}

func (r *RegistryAccess) NewComponentVersion(cv ocm.ComponentVersionAccess) model.ComponentVersion {
	return &ComponentVersion{
		registryAccess:         r,
		componentVersionAccess: cv,
	}
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (_ model.ComponentVersion, rerr error) {
	spec, err := r.octx.RepositorySpecForConfig(cdRef.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
	if err != nil {
		return nil, err
	}

	var cv ocm.ComponentVersionAccess
	// check if repository context from inline component descriptor should be used
	if r.inlineRepository != nil && reflect.DeepEqual(spec, r.inlineSpec) {
		cv, err = r.session.LookupComponentVersion(r.resolver, cdRef.ComponentName, cdRef.Version)
	} else {
		var repo ocm.Repository
		repo, err = r.session.LookupRepository(r.octx, spec)
		if err != nil {
			return nil, err
		}

		cv, err = r.session.LookupComponentVersion(repo, cdRef.ComponentName, cdRef.Version)
	}
	if err != nil {
		return nil, err
	}
	return r.NewComponentVersion(cv), nil
}

func (r *RegistryAccess) Close() error {
	err := r.session.Close()
	if err != nil {
		return err
	}
	return nil
}
