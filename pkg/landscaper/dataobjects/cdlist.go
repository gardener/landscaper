// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ComponentDescriptorList is the internal representation of a list of targets.
type ComponentDescriptorList struct {
	ComponentDescriptors []*ComponentDescriptor
}

// NewComponentDescriptorList creates a new internal component descriptor list.
func NewComponentDescriptorList() *ComponentDescriptorList {
	return &ComponentDescriptorList{
		ComponentDescriptors: []*ComponentDescriptor{},
	}
}

// NewComponentDescriptorListWithSize creates a new internal component descriptor list with a given size.
func NewComponentDescriptorListWithSize(size int) *ComponentDescriptorList {
	return &ComponentDescriptorList{
		ComponentDescriptors: make([]*ComponentDescriptor, size),
	}
}

// GetData returns the component descriptor list as an internal go map.
func (cdl *ComponentDescriptorList) GetData() (interface{}, error) {
	rawCDs := make([]cdv2.ComponentDescriptor, len(cdl.ComponentDescriptors))
	for i := range cdl.ComponentDescriptors {
		rawCDs[i] = *cdl.ComponentDescriptors[i].Descriptor
	}

	compDescList := cdv2.ComponentDescriptorList{
		Metadata: cdv2.Metadata{
			Version: cdv2.SchemaVersion,
		},
		Components: rawCDs,
	}
	raw, err := json.Marshal(compDescList)
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}
