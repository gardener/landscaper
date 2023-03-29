package cnudie

import (
	"fmt"
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

func (c *ComponentVersion) GetResource(name string, identity map[string]string) (model.Resource, error) {
	resources, err := c.componentDescriptor.GetResourcesByName(name, v2.Identity(identity))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resource with name %s and identity %v", name, identity)
	}

	return newResource(&resources[0]), nil
}
