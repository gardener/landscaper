package ocmfacade

import (
	"context"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"
	"io"
)

type Resource struct {
	resourceAccess ocm.ResourceAccess
}

func (r *Resource) GetName() string {
	return r.resourceAccess.Meta().GetName()
}

func (r *Resource) GetVersion() string {
	return r.resourceAccess.Meta().GetVersion()
}

func (r *Resource) GetType() string {
	return r.resourceAccess.Meta().GetType()
}

func (r *Resource) GetAccessType() string {
	spec, err := r.resourceAccess.Access()
	if err != nil {
		//return err
	}
	return spec.GetType()
}

func (r *Resource) GetResource() (*types.Resource, error) {
	spec := r.resourceAccess.Meta()
	data, err := runtime.DefaultYAMLEncoding.Marshal(spec)
	if err != nil {
		return nil, err
	}

	lsspec := types.Resource{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lsspec)
	if err != nil {
		return nil, err
	}

	return &lsspec, err
}

func (r *Resource) GetBlob(ctx context.Context, writer io.Writer) (rblobinfo *types.BlobInfo, rerr error) {
	accessMethod, err := r.resourceAccess.AccessMethod()
	if err != nil {
		return nil, err
	}
	//defer errors.PropagateError(&rerr, accessMethod.Close) wait for new ocm pr

	blob, err := accessMethod.Get()
	if err != nil {
		return nil, err
	}
	_, err = writer.Write(blob)
	if err != nil {
		return nil, err
	}

	blobAccess := accessio.BlobAccessForDataAccess(accessio.BLOB_UNKNOWN_DIGEST, accessio.BLOB_UNKNOWN_SIZE, accessMethod.MimeType(), accessMethod)

	return &types.BlobInfo{
		MediaType: accessMethod.MimeType(),
		Digest:    blobAccess.Digest().String(),
		Size:      blobAccess.Size(),
	}, nil
}

func (r *Resource) GetBlobInfo(ctx context.Context) (rblobinfo *types.BlobInfo, rerr error) {
	accessMethod, err := r.resourceAccess.AccessMethod()
	if err != nil {
		return nil, err
	}
	//defer errors.PropagateError(&rerr, accessMethod.Close) wait for new ocm pr

	blobAccess := accessio.BlobAccessForDataAccess(accessio.BLOB_UNKNOWN_DIGEST, accessio.BLOB_UNKNOWN_SIZE, accessMethod.MimeType(), accessMethod)

	return &types.BlobInfo{
		MediaType: accessMethod.MimeType(),
		Digest:    blobAccess.Digest().String(),
		Size:      blobAccess.Size(),
	}, nil
}
