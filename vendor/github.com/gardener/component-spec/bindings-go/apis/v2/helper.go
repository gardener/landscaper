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

// GetResource returns a external or local resource with the given type, name and version.
func (c ComponentDescriptor) GetResource(rtype, name, version string) (Resource, error) {
	res, err := c.GetLocalResource(rtype, name, version)
	if err == nil {
		return res, nil
	}

	return c.GetExternalResource(rtype, name, version)
}

// GetExternalResource returns a external resource with the given type, name and version.
func (c ComponentDescriptor) GetExternalResource(rtype, name, version string) (Resource, error) {
	for _, res := range c.ExternalResources {
		if res.GetType() == rtype && res.GetName() == name && res.GetVersion() == version {
			return res, nil
		}
	}
	return Resource{}, NotFound
}

// GetLocalResource returns a local resource with the given type, name and version.
func (c ComponentDescriptor) GetLocalResource(rtype, name, version string) (Resource, error) {
	for _, res := range c.LocalResources {
		if res.GetType() == rtype && res.GetName() == name && res.GetVersion() == version {
			return res, nil
		}
	}
	return Resource{}, NotFound
}

// GetResourcesByType returns all local and external resources of a specific resource type.
func (c ComponentDescriptor) GetResourcesByType(rtype string) []Resource {
	return append(c.GetLocalResourcesByType(rtype), c.GetLocalResourcesByType(rtype)...)
}

// GetLocalResourcesByType returns all local resources of a specific resource type.
func (c ComponentDescriptor) GetLocalResourcesByType(rtype string) []Resource {
	return getResourcesByType(c.LocalResources, rtype)
}

// GetExternalResourcesByType returns all external resources of a specific resource type.
func (c ComponentDescriptor) GetExternalResourcesByType(rtype string) []Resource {
	return getResourcesByType(c.ExternalResources, rtype)
}

func getResourcesByType(list []Resource, rtype string) []Resource {
	resources := make([]Resource, 0)
	for _, obj := range list {
		res := obj
		if res.GetType() == rtype {
			resources = append(resources, res)
		}
	}
	return resources
}

// GetResourcesByType returns all local and external resources of a specific resource type.
func (c ComponentDescriptor) GetResourcesByName(rtype, name string) []Resource {
	return append(c.GetLocalResourcesByName(rtype, name), c.GetExternalResourcesByName(rtype, name)...)
}

// GetLocalResourcesByType returns all local resources of a specific resource type.
func (c ComponentDescriptor) GetLocalResourcesByName(rtype, name string) []Resource {
	return getResourcesByName(c.LocalResources, rtype, name)
}

// GetExternalResourcesByType returns all external resources of a specific resource type.
func (c ComponentDescriptor) GetExternalResourcesByName(rtype, name string) []Resource {
	return getResourcesByName(c.ExternalResources, rtype, name)
}

func getResourcesByName(list []Resource, rtype, name string) []Resource {
	resources := make([]Resource, 0)
	for _, obj := range list {
		res := obj
		if res.GetType() == rtype && res.GetName() == name {
			resources = append(resources, res)
		}
	}
	return resources
}
