// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"

	"github.com/gardener/landscaper/pkg/components/model/types"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	_ "github.com/gardener/landscaper/pkg/components/ocmlib/repository/inline"
	_ "github.com/gardener/landscaper/pkg/components/ocmlib/repository/local"
)

type RegistryAccess struct {
	octx             ocm.Context
	session          ocm.Session
	inlineSpec       ocm.RepositorySpec
	inlineRepository ocm.Repository
	resolver         ocm.ComponentVersionResolver
}

var _ model.RegistryAccess = (*RegistryAccess)(nil)

func (r *RegistryAccess) NewComponentVersion(cv ocm.ComponentVersionAccess) (model.ComponentVersion, error) {
	if cv == nil {
		return nil, errors.New("component version access cannot be nil during facade component version creation")
	}
	// Get ocm-lib Component Descriptor
	cd := cv.GetDescriptor()

	// TODO: Remove this check
	// this is only included for compatibility reasons as the legacy ocm spec mandated component descriptors to have a
	// repository context
	if len(cd.RepositoryContexts) == 0 {
		return nil, fmt.Errorf("repository context is required")
	}
	data, err := compdesc.Encode(cd, compdesc.SchemaVersion(v2.SchemaVersion))
	if err != nil {
		return nil, err
	}

	// Create Landscaper Component Descriptor from the ocm-lib Component Descriptor
	lscd := types.ComponentDescriptor{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lscd)
	if err != nil {
		return nil, err
	}

	return &ComponentVersion{
		registryAccess:         r,
		componentVersionAccess: cv,
		componentDescriptorV2:  lscd,
	}, nil
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (_ model.ComponentVersion, rerr error) {
	if cdRef == nil {
		return nil, errors.New("component descriptor reference cannot be nil")
	}
	if cdRef.RepositoryContext == nil {
		return nil, errors.New("repository context cannot be nil")
	}

	spec, err := r.octx.RepositorySpecForConfig(cdRef.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
	if err != nil {
		return nil, err
	}

	var cv ocm.ComponentVersionAccess
	// check if repository context from inline component descriptor should be used
	if r.inlineRepository != nil && reflect.DeepEqual(spec, r.inlineSpec) {
		// in this case, resolver knows an inline repository as well as the repository specified by the repository
		// context of the inline component descriptor
		cv, err = r.session.LookupComponentVersion(r.resolver, cdRef.ComponentName, cdRef.Version)
	} else {
		// if there is no inline repository or the repository context is different from the one specified in the inline
		// component descriptor, we need to look up the repository specified by the component descriptor reference
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
	return r.NewComponentVersion(cv)
}

func (r *RegistryAccess) Close() error {
	err := r.session.Close()
	if err != nil {
		return err
	}
	return nil
}
