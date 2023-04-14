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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
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
func ResolveToComponentDescriptorList(ctx context.Context, client ctf.ComponentResolver, cd cdv2.ComponentDescriptor, repositoryContext *cdv2.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (cdv2.ComponentDescriptorList, error) {
	cds := map[componentIdentifier]cdv2.ComponentDescriptor{} // the resolved component descriptor list will be stored in this
	cdList := cdv2.ComponentDescriptorList{}
	cdList.Metadata = cd.Metadata
	if err := resolveToComponentDescriptorListHelper(ctx, client, cd, repositoryContext, overwriter, cds); err != nil {
		return cdList, err
	}
	cdList.Components = make([]cdv2.ComponentDescriptor, len(cds))
	i := 0
	for _, cd := range cds {
		cdList.Components[i] = cd
		i++
	}

	return cdList, nil
}

type componentIdentifier struct {
	Name    string
	Version string
}

func componentIdentifierFromCD(cd cdv2.ComponentDescriptor) componentIdentifier {
	return componentIdentifier{Name: cd.Name, Version: cd.Version}
}

// resolveToComponentDescriptorListHelper is a helper function which fetches all referenced component descriptor, including the referencing one
// the fetched CDs are stored in the given 'cds' map to avoid duplicates
func resolveToComponentDescriptorListHelper(ctx context.Context, client ctf.ComponentResolver, cd cdv2.ComponentDescriptor, repositoryContext *cdv2.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter, cds map[componentIdentifier]cdv2.ComponentDescriptor) error {
	cid := componentIdentifierFromCD(cd)
	if _, ok := cds[cid]; ok {
		// we have already handled this component before, no need to do it again
		return nil
	}
	cds[cid] = cd

	if len(cd.RepositoryContexts) == 0 {
		return errors.New("component descriptor must at least contain one repository context with a base url")
	}

	for _, compRef := range cd.ComponentReferences {
		resolvedComponent, err := ResolveWithOverwriter(ctx, client, repositoryContext, compRef.ComponentName, compRef.Version, overwriter)
		if err != nil {
			return fmt.Errorf("unable to resolve component descriptor for %s with version %s: %w", compRef.Name, compRef.Version, err)
		}
		err = resolveToComponentDescriptorListHelper(ctx, client, *resolvedComponent, repositoryContext, overwriter, cds)
		if err != nil {
			return fmt.Errorf("unable to resolve component references for component descriptor %s with version %s: %w", compRef.Name, compRef.Version, err)
		}
	}

	return nil
}

// ResolveWithOverwriter is like resolver.Resolve, but applies the given overwrites first.
func ResolveWithOverwriter(ctx context.Context, resolver ctf.ComponentResolver, repositoryContext *cdv2.UnstructuredTypedObject, name, version string, overwriter componentoverwrites.Overwriter) (*cdv2.ComponentDescriptor, error) {
	cdRef := &lsv1alpha1.ComponentDescriptorReference{
		RepositoryContext: repositoryContext,
		ComponentName:     name,
		Version:           version,
	}
	return ResolveWithOverwriterFromReference(ctx, resolver, cdRef, overwriter)
}

// ResolveWithOverwriterFromReference is like resolver.Resolve, but applies the given overwrites first.
func ResolveWithOverwriterFromReference(ctx context.Context, resolver ctf.ComponentResolver, cdRef *lsv1alpha1.ComponentDescriptorReference, overwriter componentoverwrites.Overwriter) (*cdv2.ComponentDescriptor, error) {
	if overwriter != nil {
		overwriter.Replace(cdRef)
	}
	return resolver.Resolve(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
}

// ResolveWithBlobResolverWithOverwriter is like resolver.ResolveWithBlobResolver, but applies the given overwrites first.
func ResolveWithBlobResolverWithOverwriter(ctx context.Context, resolver ctf.ComponentResolver, repositoryContext *cdv2.UnstructuredTypedObject, name, version string, overwriter componentoverwrites.Overwriter) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	cdRef := &lsv1alpha1.ComponentDescriptorReference{
		RepositoryContext: repositoryContext,
		ComponentName:     name,
		Version:           version,
	}
	return ResolveWithBlobResolverWithOverwriterFromReference(ctx, resolver, cdRef, overwriter)
}

// ResolveWithBlobResolverWithOverwriterFromReference is like resolver.ResolveWithBlobResolver, but applies the given overwrites first.
func ResolveWithBlobResolverWithOverwriterFromReference(ctx context.Context, resolver ctf.ComponentResolver, cdRef *lsv1alpha1.ComponentDescriptorReference, overwriter componentoverwrites.Overwriter) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	if overwriter != nil {
		overwriter.Replace(cdRef)
	}
	return resolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
}
