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

package helper

import (
	"k8s.io/apimachinery/pkg/util/sets"

	landscaperv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// GetUsedReferencedSchemes returns all used types of schema
func GetUsedReferencedSchemes(scheme *landscaperv1alpha1.JSONSchemaProps) sets.String {
	refs := sets.NewString()

	if scheme.Ref != nil {
		refs.Insert(*scheme.Ref)
	}

	if scheme.Not != nil {
		refs = refs.Union(GetUsedReferencedSchemes(scheme.Not))
	}

	// bool
	if scheme.AdditionalProperties != nil && scheme.AdditionalProperties.Schema != nil {
		refs = refs.Union(GetUsedReferencedSchemes(scheme.AdditionalProperties.Schema))
	}
	if scheme.AdditionalItems != nil && scheme.AdditionalItems.Schema != nil {
		refs = refs.Union(GetUsedReferencedSchemes(scheme.AdditionalItems.Schema))
	}

	// map
	if scheme.Properties != nil {
		for _, props := range scheme.Properties {
			refs = refs.Union(GetUsedReferencedSchemes(&props))
		}
	}
	if scheme.PatternProperties != nil {
		for _, props := range scheme.PatternProperties {
			refs = refs.Union(GetUsedReferencedSchemes(&props))
		}
	}
	if scheme.Definitions != nil {
		for _, props := range scheme.Definitions {
			refs = refs.Union(GetUsedReferencedSchemes(&props))
		}
	}

	// array
	if scheme.AllOf != nil {
		for _, props := range scheme.AllOf {
			refs = refs.Union(GetUsedReferencedSchemes(&props))
		}
	}
	if scheme.OneOf != nil {
		for _, props := range scheme.OneOf {
			refs = refs.Union(GetUsedReferencedSchemes(&props))
		}
	}
	if scheme.AnyOf != nil {
		for _, props := range scheme.AnyOf {
			refs = refs.Union(GetUsedReferencedSchemes(&props))
		}
	}

	// schema or array
	if scheme.Items != nil {
		if scheme.Items.Schema != nil {
			refs = refs.Union(GetUsedReferencedSchemes(scheme.AdditionalItems.Schema))
		}
		if len(scheme.Items.JSONSchemas) != 0 {
			for _, props := range scheme.Items.JSONSchemas {
				refs = refs.Union(GetUsedReferencedSchemes(&props))
			}
		}
	}
	return refs
}
