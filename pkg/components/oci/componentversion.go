package oci

import (
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/gardener/landscaper/pkg/components/model"
)

type OCIComponentVersion struct {
	registry            *OCIRegistry
	componentDescriptor *v2.ComponentDescriptor
	blobResolver        ctf.BlobResolver
}

var _ model.ComponentVersion = &OCIComponentVersion{}

func newOCIComponentVersion(registry *OCIRegistry, cd *v2.ComponentDescriptor, blobResolver ctf.BlobResolver) model.ComponentVersion {
	return &OCIComponentVersion{
		registry:            registry,
		componentDescriptor: cd,
		blobResolver:        blobResolver,
	}
}

func (O OCIComponentVersion) GetDescriptor() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (O OCIComponentVersion) GetDependency(name string) (model.ComponentVersion, error) {
	//TODO implement me
	panic("implement me")
}

func (O OCIComponentVersion) GetResource(name string) (model.Resource, error) {
	//TODO implement me
	panic("implement me")
}
