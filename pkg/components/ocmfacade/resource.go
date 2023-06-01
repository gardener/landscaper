package ocmfacade

import (
	"context"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"io"
)

type Resource struct {
	_ResourceAccess ocm.ResourceAccess
}

func (r Resource) GetName() string {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetVersion() string {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetType() string {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetAccessType() string {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetResource() *types.Resource {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetBlob(ctx context.Context, writer io.Writer) (*types.BlobInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetBlobInfo(ctx context.Context) (*types.BlobInfo, error) {
	//TODO implement me
	panic("implement me")
}
