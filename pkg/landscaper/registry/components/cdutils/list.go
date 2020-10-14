// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"context"
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

// ResolveEffectiveComponentDescriptor transitively resolves all referenced components of a component descriptor and
// return a list containing all resolved component descriptors.
func ResolveEffectiveComponentDescriptor(ctx context.Context, client componentsregistry.Registry, cd cdv2.ComponentDescriptor) (ResolvedComponentDescriptor, error) {
	if len(cd.RepositoryContexts) == 0 {
		return ResolvedComponentDescriptor{}, errors.New("component descriptor must at least contain one repository context with a base url")
	}
	repoCtx := cd.RepositoryContexts[len(cd.RepositoryContexts)-1]
	return ConvertFromComponentDescriptor(cd, func(ref cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
		cd, err := client.Resolve(ctx, repoCtx, ComponentReferenceToObjectMeta(ref))
		if err != nil {
			return cdv2.ComponentDescriptor{}, fmt.Errorf("unable to resolve component descriptor for %s with version %s: %w", ref.Name, ref.Version, err)
		}
		return *cd, nil
	})
}

// ResolvedComponentDescriptorList contains a map of mapped component description.
type ResolvedComponentDescriptorList struct {
	// Metadata specifies the schema version of the component.
	Metadata cdv2.Metadata `json:"meta"`
	// Components contains a map of mapped component descriptor.
	Components map[string]ResolvedComponentDescriptor
}

// ConvertFromComponentDescriptorList converts a component descriptor list to a mapped component descriptor list.
func ConvertFromComponentDescriptorList(list cdv2.ComponentDescriptorList) (ResolvedComponentDescriptorList, error) {
	mList := ResolvedComponentDescriptorList{}
	mList.Metadata = list.Metadata
	mList.Components = make(map[string]ResolvedComponentDescriptor, len(list.Components))

	refFunc := func(ref cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
		cd, err := list.GetComponent(ref.GetName(), ref.GetVersion())
		if err != nil {
			return cdv2.ComponentDescriptor{}, fmt.Errorf("component %s:%s cannot be resolved: %w", ref.GetName(), ref.GetVersion(), err)
		}
		return cd, nil
	}

	for _, cd := range list.Components {
		// todo: maybe also use version as there could be 2 components with different version
		var err error
		mList.Components[cd.Name], err = ConvertFromComponentDescriptor(cd, refFunc)
		if err != nil {
			return ResolvedComponentDescriptorList{}, err
		}
	}

	return mList, nil
}
