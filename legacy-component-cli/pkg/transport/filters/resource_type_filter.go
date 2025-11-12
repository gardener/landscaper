// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"fmt"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

type ResourceTypeFilterSpec struct {
	IncludeResourceTypes []string `json:"includeResourceTypes"`
}

type resourceTypeFilter struct {
	includeResourceTypes map[string]bool
}

func (f resourceTypeFilter) Matches(cd cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	if _, ok := f.includeResourceTypes[r.Type]; ok {
		return true
	}
	return false
}

// NewResourceTypeFilter creates a new resourceTypeFilter
func NewResourceTypeFilter(spec ResourceTypeFilterSpec) (Filter, error) {
	if len(spec.IncludeResourceTypes) == 0 {
		return nil, fmt.Errorf("includeResourceTypes must not be empty")
	}

	filter := resourceTypeFilter{
		includeResourceTypes: map[string]bool{},
	}

	for _, resourceType := range spec.IncludeResourceTypes {
		filter.includeResourceTypes[resourceType] = true
	}

	return &filter, nil
}
