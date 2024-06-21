// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/v2"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type TestComponentVersion struct {
	registryAccess      model.RegistryAccess
	componentDescriptor *types.ComponentDescriptor
}

var _ model.ComponentVersion = &TestComponentVersion{}

// NewTestComponentVersionFromReader returns a ComponentVersion for test purposes.
// It cannot be used to access referenced components.
func NewTestComponentVersionFromReader(cd *types.ComponentDescriptor) model.ComponentVersion {
	return &TestComponentVersion{
		componentDescriptor: cd,
	}
}

func (c *TestComponentVersion) GetSchemaVersion() string {
	return v2.SchemaVersion
}

func (c *TestComponentVersion) GetName() string {
	return c.componentDescriptor.GetName()
}

func (c *TestComponentVersion) GetVersion() string {
	return c.componentDescriptor.GetVersion()
}

func (t *TestComponentVersion) GetComponentDescriptor() *types.ComponentDescriptor {
	return t.componentDescriptor
}

func (t *TestComponentVersion) GetRepositoryContext() *types.UnstructuredTypedObject {
	context := t.componentDescriptor.GetEffectiveRepositoryContext()
	if context == nil {
		return nil
	}
	return context
}

func (t *TestComponentVersion) GetComponentReferences() []types.ComponentReference {
	return t.componentDescriptor.ComponentReferences
}

func (t *TestComponentVersion) GetComponentReference(name string) *types.ComponentReference {
	refs := t.GetComponentReferences()

	for i := range refs {
		ref := &refs[i]
		if ref.GetName() == name {
			return ref
		}
	}

	return nil
}

func (t *TestComponentVersion) GetReferencedComponentVersion(ctx context.Context, ref *types.ComponentReference, repositoryContext *types.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (model.ComponentVersion, error) {
	if t.registryAccess == nil {
		return nil, errors.New("no registry access provided")
	}

	cdRef := &lsv1alpha1.ComponentDescriptorReference{
		RepositoryContext: repositoryContext,
		ComponentName:     ref.ComponentName,
		Version:           ref.Version,
	}

	return model.GetComponentVersionWithOverwriter(ctx, t.registryAccess, cdRef, overwriter)
}

func (t *TestComponentVersion) GetResource(name string, identity map[string]string) (model.Resource, error) {
	resources, err := t.componentDescriptor.GetResourcesByName(name, cdv2.Identity(identity))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resource with name %s and extra identities %v found", name, identity)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("there is more than one resource with name %s and extra identities %v", name, identity)
	}

	return newTestResource(&resources[0]), nil
}
