package cnudie

import (
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/pkg/components/model"
)

type ComponentVersion struct {
	registry            *Registry
	componentDescriptor *v2.ComponentDescriptor
	blobResolver        ctf.BlobResolver
}

var _ model.ComponentVersion = &ComponentVersion{}

func newComponentVersion(registry *Registry, cd *v2.ComponentDescriptor, blobResolver ctf.BlobResolver) model.ComponentVersion {
	return &ComponentVersion{
		registry:            registry,
		componentDescriptor: cd,
		blobResolver:        blobResolver,
	}
}

func (c *ComponentVersion) GetDescriptor() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ComponentVersion) GetDependency(name string) (model.ComponentVersion, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ComponentVersion) GetResource(name string, identity model.Identity) (model.Resource, error) {
	//TODO implement me
	panic("implement me")
}
