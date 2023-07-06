// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type ComponentVersion struct {
	registryAccess      *RegistryAccess
	componentDescriptor *types.ComponentDescriptor
	blobResolver        ctf.BlobResolver
}

var _ model.ComponentVersion = &ComponentVersion{}

func newComponentVersion(registryAccess *RegistryAccess, cd *types.ComponentDescriptor, blobResolver ctf.BlobResolver) model.ComponentVersion {
	return &ComponentVersion{
		registryAccess:      registryAccess,
		componentDescriptor: cd,
		blobResolver:        blobResolver,
	}
}

func (c *ComponentVersion) GetName() string {
	return c.componentDescriptor.GetName()
}

func (c *ComponentVersion) GetVersion() string {
	return c.componentDescriptor.GetVersion()
}

func (c *ComponentVersion) GetComponentDescriptor() (*types.ComponentDescriptor, error) {
	return c.componentDescriptor, nil
}

func (c *ComponentVersion) GetRepositoryContext() (*types.UnstructuredTypedObject, error) {
	context := c.componentDescriptor.GetEffectiveRepositoryContext()
	if context == nil {
		return nil, nil
	}
	return context, nil
}

func (c *ComponentVersion) GetComponentReferences() ([]types.ComponentReference, error) {
	return c.componentDescriptor.ComponentReferences, nil
}

func (c *ComponentVersion) GetComponentReference(name string) (*types.ComponentReference, error) {
	refs, err := c.GetComponentReferences()
	if err != nil {
		return nil, err
	}

	for i := range refs {
		ref := &refs[i]
		if ref.GetName() == name {
			return ref, nil
		}
	}

	return nil, nil
}

func (c *ComponentVersion) GetReferencedComponentVersion(ctx context.Context, componentRef *types.ComponentReference,
	repositoryContext *types.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (model.ComponentVersion, error) {

	cdRef := &lsv1alpha1.ComponentDescriptorReference{
		RepositoryContext: repositoryContext,
		ComponentName:     componentRef.ComponentName,
		Version:           componentRef.Version,
	}

	return model.GetComponentVersionWithOverwriter(ctx, c.registryAccess, cdRef, overwriter)
}

func (c *ComponentVersion) GetResource(name string, selectors map[string]string) (model.Resource, error) {
	resources, err := c.componentDescriptor.GetResourcesByName(name, cdv2.Identity(selectors))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resource with name %s and extra identities %v found", name, selectors)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("there is more than one resource with name %s and extra identities %v", name, selectors)
	}

	return NewResource(&resources[0], c.blobResolver), nil
}

func (c *ComponentVersion) GetBlobResolver() (model.BlobResolver, error) {
	return c.blobResolver, nil
}
