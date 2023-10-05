// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type RegistryAccess struct {
	componentResolver       ctf.ComponentResolver
	additionalBlobResolvers []ctf.TypedBlobResolver
}

var _ model.RegistryAccess = &RegistryAccess{}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	if cdRef == nil {
		return nil, errors.New("component descriptor reference cannot be nil")
	}
	if cdRef.RepositoryContext == nil {
		return nil, errors.New("repository context cannot be nil")
	}

	cd, blobResolver, err := r.componentResolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}

	for i := range r.additionalBlobResolvers {
		additionalBlobResolver := r.additionalBlobResolvers[i]
		blobResolver, err = ctf.AggregateBlobResolvers(blobResolver, additionalBlobResolver)
		if err != nil {
			return nil, fmt.Errorf("unable to aggregate blob resolvers: %w", err)
		}
	}

	return newComponentVersion(r, cd, blobResolver), nil
}
