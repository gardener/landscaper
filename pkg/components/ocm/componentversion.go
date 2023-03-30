package ocm

import (
	"context"
	"fmt"
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

type ComponentVersion struct {
	registry *RegistryAccess
	compvers ocm.ComponentVersionAccess
}

var _ model.ComponentVersion = &ComponentVersion{}

func newComponentVersion(registry *RegistryAccess, compvers ocm.ComponentVersionAccess) model.ComponentVersion {
	return &ComponentVersion{
		registry: registry,
		compvers: compvers,
	}
}

func (c *ComponentVersion) GetName() string {
	return c.compvers.GetName()
}

func (c *ComponentVersion) GetVersion() string {
	return c.compvers.GetVersion()
}

func (c *ComponentVersion) GetRepositoryContext() []byte {
	repositoryContexts := c.compvers.GetDescriptor().RepositoryContexts
	if repositoryContexts == nil {
		return nil
	}
	rawRepositoryContext, err := repositoryContexts[len(repositoryContexts)-1].GetRaw()
	if err != nil {
		return nil
	}
	return rawRepositoryContext
}

func (c *ComponentVersion) GetDescriptor(_ context.Context) ([]byte, error) {
	return compdesc.Encode(c.compvers.GetDescriptor())
}

func (c *ComponentVersion) GetDependency(_ context.Context, name string) (model.ComponentVersion, error) {
	referencedObject, err := c.compvers.GetReference(metav1.NewIdentity(name))
	if err != nil {
		return nil, err
	}
	referencedCompName := referencedObject.GetComponentName()
	referencedCompVersion := referencedObject.

}

func (c *ComponentVersion) GetResource(name string, selectors map[string]string) (model.Resource, error) {
	resources, err := c.componentDescriptor.GetResourcesByName(name, v2.Identity(selectors))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resource with name %s and extra identities %v found", name, selectors)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("there is more than one resource with name %s and extra identities %v", name, selectors)
	}

	return newResource(&resources[0], c.blobResolver), nil
}
