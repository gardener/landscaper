package model

import (
	"errors"
	"fmt"

	"github.com/gardener/landscaper/pkg/components/model/types"
)

type ComponentVersionList struct {
	Metadata types.Metadata `json:"meta"`

	// Components contain all resolvable components with their dependencies
	Components []ComponentVersion `json:"components"`
}

func (c *ComponentVersionList) GetComponentVersion(name, version string) (ComponentVersion, error) {
	for _, comp := range c.Components {
		if comp.GetName() == name && comp.GetVersion() == version {
			return comp, nil
		}
	}
	return nil, errors.New("NotFound")
}

func (c *ComponentVersionList) GetComponentVersionByName(name string) []ComponentVersion {
	comps := make([]ComponentVersion, 0)
	for _, comp := range c.Components {
		if comp.GetName() == name {
			obj := comp
			comps = append(comps, obj)
		}
	}
	return comps
}

func ConvertComponentVersionList(componentVersionList *ComponentVersionList) (*types.ComponentDescriptorList, error) {
	if componentVersionList == nil {
		return nil, nil
	}

	components := []types.ComponentDescriptor{}

	for i := range componentVersionList.Components {
		cv := componentVersionList.Components[i]
		cd, err := cv.GetComponentDescriptor()
		if err != nil {
			return nil, fmt.Errorf("unable to get component descriptor during the conversion of a component version list")
		}
		components = append(components, *cd)
	}

	return &types.ComponentDescriptorList{
		Metadata:   componentVersionList.Metadata,
		Components: components,
	}, nil
}
