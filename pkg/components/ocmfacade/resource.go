package ocmfacade

import (
	"context"
	"github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/registries"
	_ "github.com/gardener/landscaper/pkg/components/ocmfacade/resourcetypehandlers"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"io"
	"path/filepath"
)

type Resource struct {
	resourceAccess  ocm.ResourceAccess
	handlerRegistry *registries.ResourceHandlerRegistry
}

func NewResource(access ocm.ResourceAccess) model.Resource {
	return &Resource{
		resourceAccess:  access,
		handlerRegistry: registries.Registry,
	}
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
		return ""
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

func (r *Resource) GetBlob(ctx context.Context, writer io.Writer) (_ *types.BlobInfo, rerr error) {
	accessMethod, err := r.resourceAccess.AccessMethod()
	if err != nil {
		return nil, err
	}
	defer errors.PropagateError(&rerr, accessMethod.Close)

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

func (r *Resource) GetBlobNew(ctx context.Context) (*model.TypedResourceContent, error) {
	res, err := blueprint.GetBlueprintStore().Get(ctx, r.GetCachingIdentity(ctx))
	if err != nil {
		return nil, err
	}
	if res != nil {
		return &model.TypedResourceContent{
			Type:     r.GetType(),
			Resource: res,
		}, nil
	}

	fs := memoryfs.New()
	_, _, err = download.DefaultRegistry.Download(nil, r.resourceAccess, filepath.Join("/"), fs)
	if err != nil {
		return nil, err
	}

	typedResourceContent, err := r.handlerRegistry.Get(r.GetType()).Prepare(ctx, fs)
	if err != nil {
		return nil, err
	}

	_, err = blueprint.GetBlueprintStore().Put(ctx, r.GetCachingIdentity(ctx), typedResourceContent)
	if err != nil {
		return nil, err
	}

	return typedResourceContent, nil
}

func (r *Resource) GetCachingIdentity(ctx context.Context) string {
	spec, err := r.resourceAccess.Access()
	if err != nil {
		return ""
	}
	return spec.GetInexpensiveContentVersionIdentity(r.resourceAccess.ComponentVersion())
}

func (r *Resource) GetBlobInfo(ctx context.Context) (*types.BlobInfo, error) {
	//accessMethod, err := r.resourceAccess.AccessMethod()
	//if err != nil {
	//	return nil, err
	//}
	//defer errors.PropagateError(&rerr, accessMethod.Close)
	//
	//blobAccess := accessio.BlobAccessForDataAccess(accessio.BLOB_UNKNOWN_DIGEST, accessio.BLOB_UNKNOWN_SIZE, accessMethod.MimeType(), accessMethod)
	//
	//return &types.BlobInfo{
	//	MediaType: accessMethod.MimeType(),
	//	Digest:    blobAccess.Digest().String(),
	//	Size:      blobAccess.Size(),
	//}, nil
	accessSpec, err := r.resourceAccess.Access()
	if err != nil {
		return nil, err
	}
	id := accessSpec.GetInexpensiveContentVersionIdentity(r.resourceAccess.ComponentVersion())
	return &types.BlobInfo{
		MediaType: "",
		Digest:    id,
		Size:      -1,
	}, nil

}
