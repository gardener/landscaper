// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"context"
	"errors"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
)

// BlobResolverFunc describes a helper blob resolver that implements the ctf.BlobResolver interface.
type BlobResolverFunc struct {
	info       func(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error)
	resolve    func(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error)
	canResolve func(resource cdv2.Resource) bool
}

// NewBlobResolverFunc creates a new generic blob resolver with a minimal resolve function.
func NewBlobResolverFunc(resolve func(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error)) *BlobResolverFunc {
	return &BlobResolverFunc{
		resolve: resolve,
	}
}

func (b *BlobResolverFunc) WithInfo(f func(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error)) *BlobResolverFunc {
	b.info = f
	return b
}

func (b *BlobResolverFunc) WithCanResolve(f func(resource cdv2.Resource) bool) *BlobResolverFunc {
	b.canResolve = f
	return b
}

func (b BlobResolverFunc) CanResolve(resource cdv2.Resource) bool {
	if b.canResolve == nil {
		return true
	}
	return b.canResolve(resource)
}

func (b BlobResolverFunc) Info(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error) {
	if b.info == nil {
		return b.resolve(ctx, res, nil)
	}
	return b.info(ctx, res)
}

func (b BlobResolverFunc) Resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	return b.resolve(ctx, res, writer)
}

var _ ctf.TypedBlobResolver = &BlobResolverFunc{}

// ResolveToComponentDescriptorList transitively resolves all referenced components of a component descriptor and
// return a list containing all resolved component descriptors.
func ResolveToComponentDescriptorList(ctx context.Context, client ctf.ComponentResolver, cd cdv2.ComponentDescriptor, repositoryContext *cdv2.UnstructuredTypedObject) (cdv2.ComponentDescriptorList, error) {
	cdList := cdv2.ComponentDescriptorList{}
	cdList.Metadata = cd.Metadata
	if len(cd.RepositoryContexts) == 0 {
		return cdList, errors.New("component descriptor must at least contain one repository context with a base url")
	}
	cdList.Components = []cdv2.ComponentDescriptor{cd}

	for _, compRef := range cd.ComponentReferences {
		resolvedComponent, err := client.Resolve(ctx, repositoryContext, compRef.ComponentName, compRef.Version)
		if err != nil {
			return cdList, fmt.Errorf("unable to resolve component descriptor for %s with version %s: %w", compRef.Name, compRef.Version, err)
		}
		cdList.Components = append(cdList.Components, *resolvedComponent)
		resolvedComponentReferences, err := ResolveToComponentDescriptorList(ctx, client, *resolvedComponent, repositoryContext)
		if err != nil {
			return cdList, fmt.Errorf("unable to resolve component references for component descriptor %s with version %s: %w", compRef.Name, compRef.Version, err)
		}
		cdList.Components = append(cdList.Components, resolvedComponentReferences.Components...)
	}
	return cdList, nil
}
