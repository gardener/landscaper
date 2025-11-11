// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"fmt"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

type AccessTypeFilterSpec struct {
	IncludeAccessTypes []string `json:"includeAccessTypes"`
}

type accessTypeFilter struct {
	includeAccessTypes map[string]bool
}

func (f accessTypeFilter) Matches(cd cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	if _, ok := f.includeAccessTypes[r.Access.Type]; ok {
		return true
	}
	return false
}

// NewAccessTypeFilter creates a new accessTypeFilter
func NewAccessTypeFilter(spec AccessTypeFilterSpec) (Filter, error) {
	if len(spec.IncludeAccessTypes) == 0 {
		return nil, fmt.Errorf("includeAccessTypes must not be empty")
	}

	filter := accessTypeFilter{
		includeAccessTypes: map[string]bool{},
	}

	for _, resourceType := range spec.IncludeAccessTypes {
		filter.includeAccessTypes[resourceType] = true
	}

	return &filter, nil
}
