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

// GetData returns the targets as list of internal go maps.
func (cdl *ComponentDescriptorList) GetData() ([]interface{}, error) {
	rawCDs := make([]cdv2.ComponentDescriptor, len(cdl.ComponentDescriptors))
	for i := range cdl.ComponentDescriptors {
		rawCDs[i] = *cdl.ComponentDescriptors[i].Descriptor
	}
	raw, err := json.Marshal(rawCDs)
	if err != nil {
		return nil, err
	}
	var data []interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}
