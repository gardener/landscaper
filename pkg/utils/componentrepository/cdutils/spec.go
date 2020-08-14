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

import cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

// MappedComponentDescriptor defines a landscaper internal representation of the component descriptor.
// It is only used for templating and easier access of resources.
type MappedComponentDescriptor struct {
	// Metadata specifies the schema version of the component.
	Metadata cdv2.Metadata `json:"meta"`
	// Spec contains the specification of the component.
	MappedComponentSpec `json:"component"`
}

// MappedComponentSpec defines a component spec with mapped resources instead of arrays.
type MappedComponentSpec struct {
	cdv2.ObjectMeta `json:",inline"`
	// RepositoryContexts defines the previous repositories of the component
	RepositoryContexts []cdv2.RepositoryContext `json:"repositoryContexts"`
	// Provider defines the provider type of a component.
	// It can be external or internal.
	Provider cdv2.ProviderType `json:"provider"`
	// Sources defines sources that produced the component
	Sources map[string]cdv2.Resource `json:"sources"`
	// ComponentReferences references component dependencies that can be resolved in the current context.
	ComponentReferences map[string]cdv2.ObjectMeta `json:"componentReferences"`
	// LocalResources defines internal resources that are created by the component
	LocalResources map[string]cdv2.Resource `json:"localResources"`
	// ExternalResources defines external resources that are not produced by a third party.
	ExternalResources map[string]cdv2.Resource `json:"externalResources"`
}

// ConvertFromComponentDescriptor converts a component descriptor to a mapped component descriptor.
func ConvertFromComponentDescriptor(cd cdv2.ComponentDescriptor) MappedComponentDescriptor {
	mcd := MappedComponentDescriptor{}
	mcd.ObjectMeta = cd.ObjectMeta
	mcd.RepositoryContexts = cd.RepositoryContexts
	mcd.Provider = cd.Provider

	mcd.Sources = ResourceListToMap(cd.Sources)
	mcd.ComponentReferences = ObjectMetaToMap(cd.ComponentReferences)
	mcd.LocalResources = ResourceListToMap(cd.LocalResources)
	mcd.ExternalResources = ResourceListToMap(cd.ExternalResources)

	return mcd
}

// ResourceListToMap converts a list of resources to a map of resources with the resources name as its key.
func ResourceListToMap(list []cdv2.Resource) map[string]cdv2.Resource {
	m := make(map[string]cdv2.Resource, len(list))
	for _, res := range list {
		m[res.GetName()] = res
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
