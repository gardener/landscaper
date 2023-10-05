// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package inlinecompdesc

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type InlineCompDesc struct {
	ComponentDescriptor            *compdesc.ComponentDescriptor
	ReferencedComponentDescriptors []*InlineCompDesc
	list                           []*compdesc.ComponentDescriptor
}

func New(raw []byte) (*InlineCompDesc, error) {
	cd, err := compdesc.Decode(raw)
	if err != nil {
		return nil, err
	}
	return &InlineCompDesc{
		ComponentDescriptor:            cd,
		ReferencedComponentDescriptors: []*InlineCompDesc{},
		list:                           []*compdesc.ComponentDescriptor{cd},
	}, nil
}

func NewFromInline(cd *compdesc.ComponentDescriptor) *InlineCompDesc {
	return &InlineCompDesc{
		ComponentDescriptor:            cd,
		ReferencedComponentDescriptors: []*InlineCompDesc{},
		list:                           []*compdesc.ComponentDescriptor{cd},
	}
}

// Expand takes an Inline Component Descriptor as input. Inline Component Descriptors may have Component References
// to other Inline Component Descriptors. One may wish to also define those referenced Component Descriptors inline.
// Therefore, it is possible to define the referenced Component Descriptors within the Label of the referencing
// Component Descriptor.
// This function evaluates the label to expand all Component Descriptors contained in the Inline Component Descriptor.
func (c *InlineCompDesc) Expand() error {
	return c.expand(c)
}

// expand is an auxiliary method to transparently pass the root Inline Component Descriptor. This allows to build a
// flat list of all involved Component Descriptors while traversing the references
func (c *InlineCompDesc) expand(root *InlineCompDesc) error {
	refs, err := c.ComponentDescriptor.GetComponentReferences()
	if err != nil {
		return err
	}
	for _, ref := range refs {
		label, exists := ref.GetLabels().Get(lsv1alpha1.InlineComponentDescriptorLabel)
		if exists {
			cd, err := compdesc.Decode(label)
			if err != nil {
				return err
			}
			root.list = append(root.list, cd)

			inlineCd := NewFromInline(cd)
			c.ReferencedComponentDescriptors = append(c.ReferencedComponentDescriptors, inlineCd)
			err = inlineCd.expand(root)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetFlatList returns a flat list of all the Component Descriptors aggregated by the top level Component Descriptors
// (thus, the Component References are recursively resolved), including itself.
func (c *InlineCompDesc) GetFlatList() ([]*compdesc.ComponentDescriptor, error) {
	// Check if Expand has been called yet, otherwise call it
	if len(c.ReferencedComponentDescriptors) == 0 {
		_, exists := c.ComponentDescriptor.GetLabels().Get(lsv1alpha1.InlineComponentDescriptorLabel)
		if exists {
			err := c.Expand()
			if err != nil {
				return nil, err
			}
		}
	}

	return c.list, nil
}
