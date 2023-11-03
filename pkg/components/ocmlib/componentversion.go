// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type ComponentVersion struct {
	registryAccess         *RegistryAccess
	componentVersionAccess ocm.ComponentVersionAccess
	componentDescriptorV2  cdv2.ComponentDescriptor
}

var _ model.ComponentVersion = &ComponentVersion{}

func (c *ComponentVersion) GetSchemaVersion() string {
	return c.componentVersionAccess.GetDescriptor().SchemaVersion()
}

func (c *ComponentVersion) GetName() string {
	return c.componentVersionAccess.GetName()
}

func (c *ComponentVersion) GetVersion() string {
	return c.componentVersionAccess.GetVersion()
}

func (c *ComponentVersion) GetComponentDescriptor() *types.ComponentDescriptor {
	return &c.componentDescriptorV2
}

func (c *ComponentVersion) GetRepositoryContext() *types.UnstructuredTypedObject {
	return c.componentDescriptorV2.GetEffectiveRepositoryContext()
}

func (c *ComponentVersion) GetComponentReferences() []types.ComponentReference {
	return c.componentDescriptorV2.ComponentReferences
}

func (c *ComponentVersion) GetComponentReference(name string) *types.ComponentReference {
	refs := c.GetComponentReferences()

	for i := range refs {
		ref := &refs[i]
		if ref.GetName() == name {
			return ref
		}
	}

	return nil
}

func (c *ComponentVersion) GetReferencedComponentVersion(ctx context.Context, ref *types.ComponentReference, repositoryContext *types.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (model.ComponentVersion, error) {
	if ref == nil {
		return nil, errors.New("component reference cannot be nil")
	}

	cdRef := &lsv1alpha1.ComponentDescriptorReference{
		RepositoryContext: repositoryContext,
		ComponentName:     ref.ComponentName,
		Version:           ref.Version,
	}

	return model.GetComponentVersionWithOverwriter(ctx, c.registryAccess, cdRef, overwriter)

}

func (c *ComponentVersion) GetResource(name string, identity map[string]string) (model.Resource, error) {
	resources, err := c.componentVersionAccess.GetResourcesByName(name, cdv2.Identity(identity))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resource with name %s and extra identities %v found", name, identity)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("there is more than one resource with name %s and extra identities %v", name, identity)
	}

	return NewResource(resources[0]), nil
}
