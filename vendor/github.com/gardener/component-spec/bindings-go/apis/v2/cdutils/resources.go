// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// GetImageReferenceFromList returns the image reference of a resource with the given name from a component descriptor.
// If the component name and resource name is not unique the first found object is used.
func GetImageReferenceFromList(cdList *cdv2.ComponentDescriptorList, componentName, resourceName string) (string, error) {
	components := cdList.GetComponentByName(componentName)
	if len(components) == 0 {
		return "", fmt.Errorf("no copmponent with name %q found", componentName)
	}
	cd := components[0]
	return GetImageReferenceByName(&cd, resourceName)
}

// GetImageReferenceByName returns the image reference of a resource with the given name from a component descriptor.
func GetImageReferenceByName(cd *cdv2.ComponentDescriptor, name string) (string, error) {
	resources, err := cd.GetResourcesByName(name)
	if err != nil {
		return "", err
	}
	res := resources[0]
	if res.Access.GetType() != cdv2.OCIRegistryType {
		return "", fmt.Errorf("resource is expected to be of type %q but is of type %q", cdv2.OCIRegistryType, res.Access.GetType())
	}

	data, err := res.Access.GetData()
	if err != nil {
		return "", err
	}
	ociImageAccess := &cdv2.OCIRegistryAccess{}
	if err := cdv2.NewDefaultCodec().Decode(data, ociImageAccess); err != nil {
		return "", err
	}
	return ociImageAccess.ImageReference, nil
}
