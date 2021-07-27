// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"context"
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
)

// ResolveToComponentDescriptorList transitively resolves all referenced components of a component descriptor and
// return a list containing all resolved component descriptors.
func ResolveToComponentDescriptorList(ctx context.Context, client ctf.ComponentResolver, cd cdv2.ComponentDescriptor) (cdv2.ComponentDescriptorList, error) {
	cdList := cdv2.ComponentDescriptorList{}
	cdList.Metadata = cd.Metadata
	if len(cd.RepositoryContexts) == 0 {
		return cdList, errors.New("component descriptor must at least contain one repository context with a base url")
	}
	repoCtx := cd.RepositoryContexts[len(cd.RepositoryContexts)-1]
	cdList.Components = []cdv2.ComponentDescriptor{cd}

	for _, compRef := range cd.ComponentReferences {
		resolvedComponent, err := client.Resolve(ctx, repoCtx, compRef.ComponentName, compRef.Version)
		if err != nil {
			return cdList, fmt.Errorf("unable to resolve component descriptor for %s with version %s: %w", compRef.Name, compRef.Version, err)
		}
		cdList.Components = append(cdList.Components, *resolvedComponent)
		resolvedComponentReferences, err := ResolveToComponentDescriptorList(ctx, client, *resolvedComponent)
		if err != nil {
			return cdList, fmt.Errorf("unable to resolve component references for component descriptor %s with version %s: %w", compRef.Name, compRef.Version, err)
		}
		cdList.Components = append(cdList.Components, resolvedComponentReferences.Components...)
	}
	return cdList, nil
}
