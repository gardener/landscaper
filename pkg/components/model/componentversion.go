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
	// GetSchemaVersion return the used ocm schema version.
	GetSchemaVersion() string

	// GetName returns the name of the component version.
	GetName() string

	// GetVersion returns the version of the component version
	GetVersion() string

	// GetComponentDescriptor returns the component descriptor structure as *types.ComponentDescriptor.
	// Cannot be nil
	GetComponentDescriptor() *types.ComponentDescriptor

	// GetRepositoryContext return the current repository context,
	// i.e. the last entry in the list of repository contexts.
	// TODO: Remove this method
	// ocm-spec specifies that the Repository Context is supposed to be informational about the transport history. The
	// spec does not mandate to set this property and therefore, we should not program against it.
	// Cannot be nil as component versions without repository context cannot be created (for now).
	GetRepositoryContext() *types.UnstructuredTypedObject

	// GetComponentReferences returns the list of component references of the present component version.
	// (not transitively; only the references of the present component version)
	GetComponentReferences() []types.ComponentReference

	// GetComponentReference returns the component reference with the given name.
	// Note:
	// - the name is the name of the reference, not the name of the referenced component version;
	// - the returned component reference is an entry of the present component descriptor, not the referenced
	//   component version.
	// Returns nil if there is no component reference with the given name.
	GetComponentReference(name string) *types.ComponentReference

	// GetReferencedComponentVersion returns the referenced component version
	// Cannot be nil
	GetReferencedComponentVersion(ctx context.Context, ref *types.ComponentReference, repositoryContext *types.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (ComponentVersion, error)

	// GetResource returns the resource with the given name.
	// Returns an error if there is no such resource, or more than one.
	// Currently, the Landscaper does not use the identity argument.
	GetResource(name string, identity map[string]string) (Resource, error)
}

// GetComponentDescriptor returns the component descriptor structure.
// Same as method GetComponentDescriptor of the ComponentVersion, except that the present function can handle
// the case that the provided componentVersion is nil.
func GetComponentDescriptor(componentVersion ComponentVersion) (*types.ComponentDescriptor, error) {
	if componentVersion == nil {
		return nil, nil
	}
	return componentVersion.GetComponentDescriptor(), nil
}
