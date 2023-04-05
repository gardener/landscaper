package ocm

import (
	"context"
	"fmt"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

type ComponentVersion struct {
	compvers ocm.ComponentVersionAccess
}

// Finalizer Object
// Resolve Method - need to specify repository (although it defaults to the component versions repository)
var _ model.ComponentVersion = &ComponentVersion{}

func newComponentVersion(compvers ocm.ComponentVersionAccess) model.ComponentVersion {
	return &ComponentVersion{
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

// Zusätzlicher Parameter "Repository", da die Dependency nicht notwendigerweise im gleichen Repo liegen muss?
// Uwes' Resolve Method
// Simple and glancable implementation of GetDependency WITHOUT the capability to traverse dependency chains
func (c *ComponentVersion) GetDependency(_ context.Context, name string) (model.ComponentVersion, error) {
	referenceObject, err := c.compvers.GetReference(metav1.NewIdentity(name))
	if err != nil {
		return nil, err
	}

	componentName := referenceObject.GetComponentName()
	componentVersion := referenceObject.Version

	referencedCompvers, err := c.compvers.Repository().LookupComponentVersion(componentName, componentVersion)
	if err != nil {
		return nil, err
	}

	//referencedCompvers, err := utils.ResolveReferencePath(c.compvers,
	//	[]metav1.Identity{referenceObject.GetIdentity(c.compvers.GetDescriptor().References)}, nil)
	//if err != nil {
	//	return nil, err
	//}
	return newComponentVersion(referencedCompvers), nil
}

func (c *ComponentVersion) GetResource(name string, selectors map[string]string) (model.Resource, error) {
	resources, err := c.compvers.GetDescriptor().GetResourcesByName(name, metav1.Identity(selectors))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resourceAccess with name %s and extra identities %v found", name, selectors)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("there is more than one resourceAccess with name %s and extra identities %v", name, selectors)
	}
	resourceAccess := &ResourceAccess{
		BaseAccess: &BaseAccess{
			vers:   a,
			access: r.Access,
		},
		meta: r.ResourceMeta,
	}

	resources[0].ResourceMeta
	resourceAccess := c.compvers.GetResource()

	return newResource(resources[0]), nil
}

func (c *ComponentVersion) Close() {
	c.compvers.Close()
}
