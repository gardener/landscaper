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

package v2

// DefaultComponent applies defaults to a component
func DefaultComponent(component *ComponentDescriptor) error {
	if component.Sources == nil {
		component.Sources = make([]Resource, 0)
	}
	if component.ComponentReferences == nil {
		component.ComponentReferences = make([]ObjectMeta, 0)
	}
	if component.LocalResources == nil {
		component.LocalResources = make([]Resource, 0)
	}
	if component.ExternalResources == nil {
		component.ExternalResources = make([]Resource, 0)
	}

	for i, res := range component.LocalResources {
		if len(res.Version) == 0 {
			component.LocalResources[i].Version = component.GetVersion()
		}
	}
	return nil
}

func DefaultList(list *ComponentDescriptorList) error {
	for i, comp := range list.Components {
		if len(comp.Metadata.Version) == 0 {
			list.Components[i].Metadata.Version = list.Metadata.Version
		}
	}
	return nil
}
