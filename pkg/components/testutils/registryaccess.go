package testutils

import (
	"context"
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type TestRegistryAccess struct {
	componentDescriptors []types.ComponentDescriptor
	blobResolver         model.BlobResolver
}

var _ model.RegistryAccess = &TestRegistryAccess{}

// NewTestRegistryAccess creates a RegistryAccess from a list of component descriptors.
// This constructor is intended to create test objects.
func NewTestRegistryAccess(componentDescriptors ...types.ComponentDescriptor) *TestRegistryAccess {
	return &TestRegistryAccess{
		componentDescriptors: componentDescriptors,
	}
}

func (t *TestRegistryAccess) WithBlobResolver(blobResolver model.BlobResolver) *TestRegistryAccess {
	t.blobResolver = blobResolver
	return t
}

func (t *TestRegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	for i := range t.componentDescriptors {
		cd := &t.componentDescriptors[i]
		if cd.GetName() == cdRef.ComponentName && cd.GetVersion() == cdRef.Version {
			return newTestComponentVersionWithRegistryAccess(cd, t.blobResolver, t), nil
		}
	}

	return nil, fmt.Errorf("component not found: %v", cdRef)
}
