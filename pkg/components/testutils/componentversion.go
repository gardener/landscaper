package testutils

import (
	"context"
	"errors"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type TestComponentVersion struct {
	registryAccess      model.RegistryAccess
	componentDescriptor *types.ComponentDescriptor
	blobResolver        model.BlobResolver
}

var _ model.ComponentVersion = &TestComponentVersion{}

// NewTestComponentVersion returns a ComponentVersion for test purposes.
// It cannot be used to access referenced components.
func NewTestComponentVersion(cd *types.ComponentDescriptor, blobResolver model.BlobResolver) model.ComponentVersion {
	return &TestComponentVersion{
		componentDescriptor: cd,
		blobResolver:        blobResolver,
	}
}

// NewTestComponentVersionFromReader returns a ComponentVersion for test purposes.
// It cannot be used to access referenced components.
func NewTestComponentVersionFromReader(cd *types.ComponentDescriptor, reader io.Reader, info *types.BlobInfo) model.ComponentVersion {
	return &TestComponentVersion{
		componentDescriptor: cd,
		blobResolver:        newTestBlobResolverFromReader(reader, info),
	}
}

// newTestComponentVersionWithRegistryAccess returns a ComponentVersion for test purposes.
// The provided registryAccess is used to get referenced components.
func newTestComponentVersionWithRegistryAccess(cd *types.ComponentDescriptor, blobResolver model.BlobResolver, registryAccess model.RegistryAccess) model.ComponentVersion {
	return &TestComponentVersion{
		registryAccess:      registryAccess,
		componentDescriptor: cd,
		blobResolver:        blobResolver,
	}
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

	return newTestResource(&resources[0], t.blobResolver), nil
}

func (t *TestComponentVersion) GetBlobResolver() (model.BlobResolver, error) {
	return t.blobResolver, nil
}
