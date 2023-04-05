package ocm

import (
	"context"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"io"
)

type Resource struct {
	resourceAccess ocm.ResourceAccess
}

var _ model.Resource = &Resource{}

func newResource(res ocm.ResourceAccess) *Resource {
	return &Resource{
		resourceAccess: res,
	}
}

func (r Resource) GetName() string {
	return r.resourceAccess.Meta().Name
}

func (r Resource) GetVersion() string {
	return r.resourceAccess.Meta().Version
}

func (r Resource) GetDescriptor(ctx context.Context) ([]byte, error) {
	return r.resourceAccess.
}

func (r Resource) GetBlob(ctx context.Context, writer io.Writer) error {
	_, err := r.blobResolver.Resolve(ctx, *r.resourceAccess, writer)
	return err
}

func (r Resource) GetBlobInfo(ctx context.Context) (*model.BlobInfo, error) {
	info, err := r.blobResolver.Info(ctx, *r.resourceAccess)
	if err != nil {
		return nil, err
	}

	return &model.BlobInfo{
		MediaType: info.MediaType,
		Digest:    info.Digest,
	}, nil
}
