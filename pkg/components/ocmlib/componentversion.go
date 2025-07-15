// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"errors"
	"fmt"

	"ocm.software/ocm/api/ocm"
	v1 "ocm.software/ocm/api/ocm/compdesc/meta/v1"

	cdv2 "github.com/gardener/landscaper/component-spec-bindings-go/apis/v2"

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
	if len(identity) > 0 {
		return nil, fmt.Errorf("failed to get resource with name %s and extra identities %v: extra identity is not supported", name, identity)
	}

	resource, err := c.componentVersionAccess.GetResource(v1.NewIdentity(name))
	if err != nil {
		return nil, fmt.Errorf("failed to get resource with name %s", name)
	}

	return NewResource(resource), nil
}

func (c *ComponentVersion) GetOCMObject() ocm.ComponentVersionAccess {
	return c.componentVersionAccess
}
