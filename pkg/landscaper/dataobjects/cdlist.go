// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ ImportedBase = &ComponentDescriptorList{}

// ComponentDescriptorList is the internal representation of a list of targets.
type ComponentDescriptorList struct {
	ComponentDescriptors []*ComponentDescriptor
	Def                  *lsv1alpha1.ComponentDescriptorImport
}

// NewComponentDescriptorList creates a new internal component descriptor list.
func NewComponentDescriptorList() *ComponentDescriptorList {
	return NewComponentDescriptorListWithSize(0)
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

// Imported interface

func (cdl *ComponentDescriptorList) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeComponentDescriptorList
}

func (cdl *ComponentDescriptorList) IsListTypeImport() bool {
	return true
}

func (cdl *ComponentDescriptorList) GetInClusterObject() client.Object {
	return nil
}
func (cdl *ComponentDescriptorList) GetInClusterObjects() []client.Object {
	// component descriptors are not represented as in-cluster landscaper objects
	return nil
}

func (cdl *ComponentDescriptorList) ComputeConfigGeneration() string {
	return ""
}

func (cdl *ComponentDescriptorList) GetListItems() []ImportedBase {
	res := make([]ImportedBase, len(cdl.ComponentDescriptors))
	for i := range cdl.ComponentDescriptors {
		res[i] = cdl.ComponentDescriptors[i]
	}
	return res
}

func (cdl *ComponentDescriptorList) GetImportReference() string {
	return ""
}

func (cdl *ComponentDescriptorList) GetImportDefinition() interface{} {
	return cdl.Def
}
