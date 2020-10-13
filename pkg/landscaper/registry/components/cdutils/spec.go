// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// {{ .cd.componentReferences.my-comp.localReources }}

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
	// LocalResources defines internal resources that are created by the component
	LocalResources map[string]cdv2.Resource `json:"localResources"`
	// ExternalResources defines external resources that are not produced by a third party.
	ExternalResources map[string]cdv2.Resource `json:"externalResources"`
}

func (cd ResolvedComponentDescriptor) LatestRepositoryContext() cdv2.RepositoryContext {
	return cd.RepositoryContexts[len(cd.RepositoryContexts)-1]
}

type ResolveComponentReferenceFunc = func(meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error)

// ConvertFromComponentDescriptor converts a component descriptor to a resolved component descriptor.
func ConvertFromComponentDescriptor(cd cdv2.ComponentDescriptor, refFunc ResolveComponentReferenceFunc) (ResolvedComponentDescriptor, error) {
	mcd := ResolvedComponentDescriptor{}
	mcd.ObjectMeta = cd.ObjectMeta
	mcd.RepositoryContexts = cd.RepositoryContexts
	mcd.Provider = cd.Provider

	mcd.Sources = SourceListToMap(cd.Sources)
	mcd.LocalResources = ResourceListToMap(cd.LocalResources)
	mcd.ExternalResources = ResourceListToMap(cd.ExternalResources)

	var err error
	mcd.ComponentReferences, err = ResolveComponentReferences(cd.ComponentReferences, refFunc)
	return mcd, err
}

// ResolveComponentReferences resolves a list of component references to resolved components.
func ResolveComponentReferences(refs []cdv2.ComponentReference, refFunc ResolveComponentReferenceFunc) (map[string]ResolvedComponentDescriptor, error) {
	resolvedComponentRefs := map[string]ResolvedComponentDescriptor{}
	for _, ref := range refs {
		cd, err := refFunc(ref)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component: %w", err)
		}
		resolvedComponent, err := ConvertFromComponentDescriptor(cd, refFunc)
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
