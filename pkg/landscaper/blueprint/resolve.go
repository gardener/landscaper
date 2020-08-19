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

package blueprint

import (
	"context"
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/componentrepository/cdutils"
)

// Resolve returns a blueprint from a given reference.
func Resolve(ctx context.Context, op operation.Interface, reference lsv1alpha1.RemoteBlueprintReference) (*Blueprint, error) {
	cd, err := op.ComponentRepository().Resolve(ctx, reference.RepositoryContext, reference.ObjectMeta())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", reference, err)
	}

	res, err := cdutils.FindResourceInComponentByVersionedReference(*cd, lsv1alpha1.BlueprintResourceType, reference.VersionedResourceReference)
	if err != nil {
		return nil, fmt.Errorf("unable to find blueprint resource in component descriptor for ref %#v: %w", reference, err)
	}

	blue, err := op.Registry().GetBlueprint(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch blueprint for ref %#v: %w", reference, err)
	}

	blueprintContent, err := op.Registry().GetContent(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch content for ref %#v: %w", reference, err)
	}

	intBlueprint, err := New(blue, blueprintContent)
	if err != nil {
		return nil, fmt.Errorf("unable to create internal blueprint representation for ref %#v: %w", reference, err)
	}
	return intBlueprint, nil
}
