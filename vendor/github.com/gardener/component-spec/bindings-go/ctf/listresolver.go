// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ListResolver describes a ComponentResolver using a list of Component Descriptors
type ListResolver struct {
	List *cdv2.ComponentDescriptorList
	blobResolver *AggregatedBlobResolver
}

// NewListResolver creates a new list resolver.
func NewListResolver(list *cdv2.ComponentDescriptorList, resolvers ...BlobResolver) (*ListResolver, error) {
	lr := &ListResolver{
		List: list,
	}
	if len(resolvers) != 0 {
		blobResolver, err := NewAggregatedBlobResolver(resolvers...)
		if err != nil {
			return nil, err
		}
		lr.blobResolver = blobResolver
	}
	return lr, nil
}

var _ ComponentResolver = &ListResolver{}

func (l ListResolver) Resolve(_ context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, error) {
	for _, comp := range l.List.Components {
		if !cdv2.TypedObjectEqual(repoCtx, comp.GetEffectiveRepositoryContext()) {
			continue
		}
		if comp.Name == name && comp.Version == version {
			return comp.DeepCopy(), nil
		}
	}

	return nil, NotFoundError
}

func (l ListResolver) ResolveWithBlobResolver(ctx context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, BlobResolver, error) {
	cd, err := l.Resolve(ctx, repoCtx, name, version)
	if err != nil {
		return nil, nil, err
	}
	if l.blobResolver == nil {
		return nil, nil, BlobResolverNotDefinedError
	}
	return cd, l.blobResolver, err
}
