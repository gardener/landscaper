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

// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
// to OCI Image References.
type ComponentNameMapping string

const (
	OCIRegistryURLPathMapping ComponentNameMapping = "urlPath"
	OCIRegistryDigestMapping  ComponentNameMapping = "sha256-digest"
)

// OCIRegistryRepository describes a component repository backed by a oci registry.
type OCIRegistryRepository struct {
	ObjectType `json:",inline"`
	// BaseURL is the base url of the repository to resolve components.
	BaseURL string `json:"baseUrl"`
	// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
	// to OCI Image References.
	ComponentNameMapping ComponentNameMapping `json:"componentNameMapping"`
}

// NewOCIRegistryRepository creates a new OCIRegistryRepository accessor
func NewOCIRegistryRepository(baseURL string, mapping ComponentNameMapping) *OCIRegistryRepository {
	if len(mapping) == 0 {
		mapping = OCIRegistryURLPathMapping
	}
	return &OCIRegistryRepository{
		ObjectType: ObjectType{
			Type: OCIRegistryType,
		},
		BaseURL:              baseURL,
		ComponentNameMapping: mapping,
	}
}

func (a *OCIRegistryRepository) GetType() string {
	return OCIRegistryType
}
