package ocmfacade

import (
	"context"
	"fmt"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	v1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/utils"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type ComponentVersion struct {
	_ComponentVersionAccess ocm.ComponentVersionAccess
}

func (c *ComponentVersion) GetName() string {
	return c._ComponentVersionAccess.GetName()
}

func (c *ComponentVersion) GetVersion() string {
	return c._ComponentVersionAccess.GetVersion()
}

func (c *ComponentVersion) GetComponentDescriptor() (*types.ComponentDescriptor, error) {
	// Get ocm-lib Component Descriptor
	cd := c._ComponentVersionAccess.GetDescriptor()
	data, err := compdesc.Encode(cd)
	if err != nil {
		return nil, err
	}

	// Create Landscaper Component Descriptor from the ocm-lib Component Descriptor
	lscd := types.ComponentDescriptor{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lscd)
	if err != nil {
		return nil, err
	}

	return &lscd, nil
}

func (c *ComponentVersion) GetRepositoryContext() (*types.UnstructuredTypedObject, error) {
	// Get ocm-lib (effective) Repository Context
	spec := c._ComponentVersionAccess.GetDescriptor().GetEffectiveRepositoryContext()
	data, err := runtime.DefaultYAMLEncoding.Marshal(&spec)
	if err != nil {
		return nil, err
	}

	// Create Landscaper (effective) Repository Context from ocm-lib Repository Context
	lsspec := types.UnstructuredTypedObject{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lsspec)
	if err != nil {
		return nil, err
	}

	return &lsspec, err
}

func (c *ComponentVersion) GetComponentReferences() ([]types.ComponentReference, error) {
	// Get ocm-lib Component References
	refs := c._ComponentVersionAccess.GetDescriptor().References
	data, err := runtime.DefaultYAMLEncoding.Marshal(&refs)
	if err != nil {
		return nil, err
	}

	// Create Landscaper Component References from ocm-lib Component References
	lsrefs := make([]types.ComponentReference, 0, refs.Len())
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lsrefs)
	if err != nil {
		return nil, err
	}

	return lsrefs, nil
}

func (c *ComponentVersion) GetComponentReference(name string) (*types.ComponentReference, error) {
	// Get ocm-lib Component Reference by name
	refs, err := c._ComponentVersionAccess.GetDescriptor().GetReferencesByName(name)
	if err != nil {
		return nil, err
	}
	if refs.Len() != 1 {
		return nil, errors.New("given reference name is not unique within the component descriptor")
	}
	ref := refs[0]

	data, err := runtime.DefaultYAMLEncoding.Marshal(&ref)
	if err != nil {
		return nil, err
	}

	// Create Landscaper Component Reference from ocm-lib Component Reference
	lsref := types.ComponentReference{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lsref)
	if err != nil {
		return nil, err
	}

	return &lsref, nil
}

func (c *ComponentVersion) GetReferencedComponentVersion(ctx context.Context, ref *types.ComponentReference, repositoryContext *types.UnstructuredTypedObject, overwriter componentoverwrites.Overwriter) (rcompvers model.ComponentVersion, rerr error) {
	// Prepare input arguments for Resolve Reference function
	id := v1.Identity(ref.GetIdentity())
	path := []v1.Identity{id}
	repo := c._ComponentVersionAccess.Repository()
	defer errors.PropagateError(&rerr, repo.Close)

	// Resolve the given reference path (here, this is not really a path, but only a direct reference)
	compvers, err := utils.ResolveReferencePath(c._ComponentVersionAccess, path, c._ComponentVersionAccess.Repository())
	if err != nil {
		return nil, err
	}

	return &ComponentVersion{
		_ComponentVersionAccess: compvers,
	}, nil

}

func (c *ComponentVersion) GetResource(name string, identity map[string]string) (model.Resource, error) {
	resources, err := c._ComponentVersionAccess.GetResourcesByName(name, cdv2.Identity(identity))
	if err != nil {
		return nil, err
	}
	if len(resources) < 1 {
		return nil, fmt.Errorf("no resource with name %s and extra identities %v found", name, identity)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("there is more than one resource with name %s and extra identities %v", name, identity)
	}

	return &Resource{
		_ResourceAccess: resources[0],
	}, nil
}

func (c *ComponentVersion) GetBlobResolver() model.BlobResolver {
	//TODO implement me
	panic("implement me")
}
