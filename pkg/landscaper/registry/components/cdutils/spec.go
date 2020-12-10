// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
)

// ResolvedComponentDescriptor defines a landscaper internal representation of the component descriptor.
// It is only used for templating and easier access of resources.
type ResolvedComponentDescriptor struct {
	// Metadata specifies the schema version of the component.
	Metadata cdv2.Metadata `json:"meta"`
	// Spec contains the specification of the component.
	ResolvedComponentSpec `json:"component"`
}

// ResolvedComponentSpec defines a component spec with mapped resources instead of arrays.
type ResolvedComponentSpec struct {
	cdv2.ObjectMeta `json:",inline"`
	// RepositoryContexts defines the previous repositories of the component
	RepositoryContexts []cdv2.RepositoryContext `json:"repositoryContexts"`
	// Provider defines the provider type of a component.
	// It can be external or internal.
	Provider cdv2.ProviderType `json:"provider"`
	// Sources defines sources that produced the component
	Sources map[string]cdv2.Source `json:"sources"`
	// ComponentReferences references component dependencies that can be resolved in the current context.
	ComponentReferences map[string]ResolvedComponentDescriptor `json:"componentReferences"`
	// Resources defines internal and external resources that are created by the component.
	Resources map[string]cdv2.Resource `json:"resources"`
}

func (cd ResolvedComponentDescriptor) LatestRepositoryContext() cdv2.RepositoryContext {
	return cd.RepositoryContexts[len(cd.RepositoryContexts)-1]
}

type ResolveComponentReferenceFunc func(ctx context.Context, meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error)

// ComponentReferenceResolverFromList creates a component reference resolver from a list of components.
func ComponentReferenceResolverFromList(list *cdv2.ComponentDescriptorList) ResolveComponentReferenceFunc {
	return func(_ context.Context, meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
		if list == nil {
			return cdv2.ComponentDescriptor{}, cdv2.NotFound
		}
		return list.GetComponent(meta.ComponentName, meta.Version)
	}
}

// ComponentReferenceResolverFromResolver creates a component reference resolver from a ctf component resolver
func ComponentReferenceResolverFromResolver(resolver ctf.ComponentResolver, repoCtx cdv2.RepositoryContext) ResolveComponentReferenceFunc {
	return func(ctx context.Context, meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
		cd, _, err := resolver.Resolve(ctx, repoCtx, meta.ComponentName, meta.GetVersion())
		if err != nil {
			return cdv2.ComponentDescriptor{}, err
		}
		return *cd, nil
	}
}

// ConvertFromComponentDescriptor converts a component descriptor to a resolved component descriptor.
func ConvertFromComponentDescriptor(ctx context.Context, cd cdv2.ComponentDescriptor, refFunc ResolveComponentReferenceFunc) (ResolvedComponentDescriptor, error) {
	mcd := ResolvedComponentDescriptor{}
	mcd.Metadata = cd.Metadata
	mcd.ObjectMeta = cd.ObjectMeta
	mcd.RepositoryContexts = cd.RepositoryContexts
	mcd.Provider = cd.Provider

	mcd.Sources = SourceListToMap(cd.Sources)
	mcd.Resources = ResourceListToMap(cd.Resources)

	var err error
	mcd.ComponentReferences, err = ResolveComponentReferences(ctx, cd.ComponentReferences, refFunc)
	return mcd, err
}

// ResolveComponentReferences resolves a list of component references to resolved components.
func ResolveComponentReferences(ctx context.Context, refs []cdv2.ComponentReference, refFunc ResolveComponentReferenceFunc) (map[string]ResolvedComponentDescriptor, error) {
	resolvedComponentRefs := map[string]ResolvedComponentDescriptor{}
	for _, ref := range refs {
		cd, err := refFunc(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component: %w", err)
		}
		resolvedComponent, err := ConvertFromComponentDescriptor(ctx, cd, refFunc)
		if err != nil {
			return nil, fmt.Errorf("unable to convert component %s:%s in resolved component: %w", cd.Name, cd.Version, err)
		}
		resolvedComponentRefs[ref.Name] = resolvedComponent
	}
	return resolvedComponentRefs, nil
}

// ResourceListToMap converts a list of resources to a map of resources with the resources name as its key.
func ResourceListToMap(list []cdv2.Resource) map[string]cdv2.Resource {
	m := make(map[string]cdv2.Resource, len(list))
	for _, res := range list {
		m[res.GetName()] = res
	}
	return m
}

// ResourceListToMap converts a list of resources to a map of resources with the resources name as its key.
func SourceListToMap(list []cdv2.Source) map[string]cdv2.Source {
	m := make(map[string]cdv2.Source, len(list))
	for _, src := range list {
		m[src.GetName()] = src
	}
	return m
}

// ObjectMetaToMap converts a list of ObjectMeta objects to a map of ObjectMeta objects with the object name as its key.
func ObjectMetaToMap(list []cdv2.ObjectMeta) map[string]cdv2.ObjectMeta {
	m := make(map[string]cdv2.ObjectMeta, len(list))
	for _, res := range list {
		m[res.GetName()] = res
	}
	return m
}
