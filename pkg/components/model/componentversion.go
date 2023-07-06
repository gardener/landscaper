// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"

	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type ComponentVersion interface {

	// GetName returns the name of the component version.
	GetName() string

	// GetVersion returns the version of the component version
	GetVersion() string

	// GetComponentDescriptor returns the component descriptor structure as *types.ComponentDescriptor.
	GetComponentDescriptor() (*types.ComponentDescriptor, error)

	// GetRepositoryContext return the current repository context,
	// i.e. the last entry in the list of repository contexts.
	GetRepositoryContext() (*types.UnstructuredTypedObject, error)

	// GetComponentReferences returns the list of component references of the present component version.
	// (not transitively; only the references of the present component version)
	GetComponentReferences() ([]types.ComponentReference, error)

	// GetComponentReference returns the component reference with the given name.
	// Note:
	// - the name is the name of the reference, not the name of the referenced component version;
	// - the returned component reference is an entry of the present component descriptor, not the referenced
	//   component version.
	// Returns nil if there is no component reference with the given name.
	GetComponentReference(name string) (*types.ComponentReference, error)

	// GetReferencedComponentVersion returns
	GetReferencedComponentVersion(ctx context.Context, ref *types.ComponentReference, repositoryContext *types.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (ComponentVersion, error)

	// GetResource returns the resource with the given name.
	// Returns an error if there is no such resource, or more than one.
	// Currently, the Landscaper does not use the identity argument.
	GetResource(name string, identity map[string]string) (Resource, error)

	GetBlobResolver() (BlobResolver, error)
}

// GetComponentDescriptor returns the component descriptor structure.
// Same as method GetComponentDescriptor of the ComponentVersion, except that the present function can handle
// the case that the provided componentVersion is nil.
func GetComponentDescriptor(componentVersion ComponentVersion) (*types.ComponentDescriptor, error) {
	if componentVersion == nil {
		return nil, nil
	}
	return componentVersion.GetComponentDescriptor()
}
