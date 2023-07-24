// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"io"

	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/pkg/components/cnudie/registries"
	_ "github.com/gardener/landscaper/pkg/components/cnudie/resourcetypehandlers"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

func NewResource(res *types.Resource, blobResolver ctf.BlobResolver) model.Resource {
	return &Resource{
		resource:        res,
		blobResolver:    blobResolver,
		handlerRegistry: registries.Registry,
	}
}

type Resource struct {
	resource        *types.Resource
	blobResolver    ctf.BlobResolver
	handlerRegistry *registries.ResourceHandlerRegistry
}

var _ model.Resource = &Resource{}

func (r *Resource) GetName() string {
	return r.resource.GetName()
}

func (r *Resource) GetVersion() string {
	return r.resource.GetVersion()
}

func (r *Resource) GetType() string {
	return r.resource.GetType()
}

func (r *Resource) GetAccessType() string {
	return r.resource.Access.GetType()
}

func (r *Resource) GetResource() (*types.Resource, error) {
	return r.resource, nil
}

func (r *Resource) GetBlobNew(ctx context.Context) (*model.TypedResourceContent, error) {
	handler := r.handlerRegistry.Get(r.GetType())
	return handler.GetResourceContent(ctx, r, r.blobResolver)
}

func (r *Resource) GetBlob(ctx context.Context, writer io.Writer) (*types.BlobInfo, error) {
	return r.blobResolver.Resolve(ctx, *r.resource, writer)
}

func (r *Resource) GetBlobInfo(ctx context.Context) (*types.BlobInfo, error) {
	return r.blobResolver.Info(ctx, *r.resource)
}

func (r *Resource) GetCachingIdentity(ctx context.Context) string {
	blobInfo, _ := r.blobResolver.Info(ctx, *r.resource)
	if blobInfo == nil {
		return ""
	}
	return blobInfo.Digest
}

//
//func (r Resource) getBlueprint(ctx context.Context) (_ vfs.FileSystem, rerr error) {
//	vfs := memoryfs.New()
//	buffer := &bytes.Buffer{}
//	resolver := componentresolvers.BlueprintResolver{}
//	blobInfo, err := resolver.Resolve(ctx, *r.resource, buffer)
//	if err != nil {
//		return nil, err
//	}
//	mediaType, err := mediatype.Parse(blobInfo.MediaType)
//	if err != nil {
//		return nil, fmt.Errorf("unable to parse media type: %w", err)
//	}
//
//	if mediaType.String() == mediatype.MediaTypeGZip || mediaType.IsCompressed(mediatype.GZipCompression) {
//		reader, err := gzip.NewReader(buffer)
//		if err != nil {
//			if err == gzip.ErrHeader {
//				return nil, errors.New("expected a gzip compressed tar")
//			}
//			return nil, err
//		}
//		defer errors.PropagateError(&rerr, reader.Close)
//	}
//
//	if err := vfs.Mkdir(bpPath, os.ModePerm); err != nil && !os.IsExist(err) {
//		return fmt.Errorf("unable to create bluprint directory: %w", err)
//	}
//	if err := tar.ExtractTar(ctx, blobReader, fs, tar.ToPath(bpPath), tar.Overwrite(true)); err != nil {
//		return fmt.Errorf("unable to extract blueprint from blob: %w", err)
//	}
//	if err := eg.Wait(); err != nil {
//		return err
//	}
//	return nil
//}
