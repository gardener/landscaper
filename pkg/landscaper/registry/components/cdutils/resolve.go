// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cdutils

import (
	"context"
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

// ResolveEffectiveComponentDescriptorList transitively resolves all referenced components of a component descriptor and
// return a list containing all resolved component descriptors.
func ResolveEffectiveComponentDescriptorList(ctx context.Context, client componentsregistry.Registry, cd cdv2.ComponentDescriptor) (cdv2.ComponentDescriptorList, error) {
	if len(cd.RepositoryContexts) == 0 {
		return cdv2.ComponentDescriptorList{}, errors.New("component descriptor must at least contain one repository context with a base url")
	}
	var (
		repoCtx = cd.RepositoryContexts[len(cd.RepositoryContexts)-1]
		cdList  = cdv2.ComponentDescriptorList{
			Metadata:   cd.Metadata,
			Components: []cdv2.ComponentDescriptor{cd},
		}
	)

	for _, ref := range cd.ComponentReferences {
		cd, err := client.Resolve(ctx, repoCtx, ref)
		if err != nil {
			return cdv2.ComponentDescriptorList{}, fmt.Errorf("unable to resolve component descriptor for %s with version %s: %w", ref.Name, ref.Version, err)
		}

		list, err := ResolveEffectiveComponentDescriptorList(ctx, client, *cd)
		if err != nil {
			return cdv2.ComponentDescriptorList{}, fmt.Errorf("unable to resolve transitive components for %s with version %s: %w", ref.GetName(), ref.GetVersion(), err)
		}

		cdList.Components = append(cdList.Components, list.Components...)
	}

	return cdList, nil
}
