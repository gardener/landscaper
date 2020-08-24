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

// MappedComponentDescriptorList contains a map of mapped component description.
type MappedComponentDescriptorList struct {
	// Metadata specifies the schema version of the component.
	Metadata cdv2.Metadata `json:"meta"`
	// Components contains a map of mapped component descriptor.
	Components map[string]MappedComponentDescriptor
}

// ConvertFromComponentDescriptorList converts a component descriptor list to a mapped component descriptor list.
func ConvertFromComponentDescriptorList(list cdv2.ComponentDescriptorList) MappedComponentDescriptorList {
	mList := MappedComponentDescriptorList{}
	mList.Metadata = list.Metadata
	mList.Components = make(map[string]MappedComponentDescriptor, len(list.Components))

	for _, cd := range list.Components {
		// todo: maybe also use version as there could be 2 components with different version
		mList.Components[cd.Name] = ConvertFromComponentDescriptor(cd)
	}

	return mList
}
