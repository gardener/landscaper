// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

const (
	// ComponentNameFilterType defines the type of a component name filter
	ComponentNameFilterType = "ComponentNameFilter"

	// ResourceTypeFilterType defines the type of a resource type filter
	ResourceTypeFilterType = "ResourceTypeFilter"

	// AccessTypeFilterType defines the type of a access type filter
	AccessTypeFilterType = "AccessTypeFilter"
)

// NewFilterFactory creates a new filter factory
// How to add a new filter:
// - Add Go file to filters package which contains the source code of the new filter
// - Add string constant for new filter type -> will be used in FilterFactory.Create()
// - Add source code for creating new filter to FilterFactory.Create() method
func NewFilterFactory() *FilterFactory {
	return &FilterFactory{}
}

// FilterFactory defines a helper struct for creating filters
type FilterFactory struct{}

// Create creates a new filter defined by a type and a spec
func (f *FilterFactory) Create(filterType string, spec *json.RawMessage) (Filter, error) {
	switch filterType {
	case ComponentNameFilterType:
		return f.createComponentNameFilter(spec)
	case ResourceTypeFilterType:
		return f.createResourceTypeFilter(spec)
	case AccessTypeFilterType:
		return f.createAccessTypeFilter(spec)
	default:
		return nil, fmt.Errorf("unknown filter type %s", filterType)
	}
}

func (f *FilterFactory) createComponentNameFilter(rawSpec *json.RawMessage) (Filter, error) {
	var spec ComponentNameFilterSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewComponentNameFilter(spec)
}

func (f *FilterFactory) createResourceTypeFilter(rawSpec *json.RawMessage) (Filter, error) {
	var spec ResourceTypeFilterSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewResourceTypeFilter(spec)
}

func (f *FilterFactory) createAccessTypeFilter(rawSpec *json.RawMessage) (Filter, error) {
	var spec AccessTypeFilterSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewAccessTypeFilter(spec)
}
