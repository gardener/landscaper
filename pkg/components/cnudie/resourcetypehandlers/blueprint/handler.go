// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package blueprint

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/open-component-model/ocm/pkg/errors"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/components/cnudie/registries"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/tar"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

func init() {
	registries.Registry.Register(mediatype.BlueprintType, New())
	registries.Registry.Register(mediatype.OldBlueprintType, New())
}

type BlueprintHandler struct{}

func New() *BlueprintHandler {
	return &BlueprintHandler{}
}

func (h *BlueprintHandler) GetResourceContent(ctx context.Context, r model.Resource, blobResolver model.BlobResolver) (*model.TypedResourceContent, error) {
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

	buffer := new(bytes.Buffer)
	resource, err := r.GetResource()
	if err != nil {
		return nil, err
	}
	blobInfo, err := blobResolver.Resolve(ctx, *resource, buffer)
	if err != nil {
		return nil, err
	}
	typedResourceContent, err := h.Prepare(ctx, buffer, blobInfo)
	if err != nil {
		return nil, err
	}

	_, err = blueprint.GetBlueprintStore().Put(ctx, r.GetCachingIdentity(ctx), typedResourceContent)
	if err != nil {
		return nil, err
	}

	return typedResourceContent, nil
}

func (h *BlueprintHandler) Prepare(ctx context.Context, data io.Reader, info *types.BlobInfo) (_ *model.TypedResourceContent, rerr error) {
	blobReader := data

	mediaType, err := mediatype.Parse(info.MediaType)
	if err != nil {
		return nil, fmt.Errorf("unable to parse media type: %w", err)
	}
	if mediaType.String() == mediatype.MediaTypeGZip || mediaType.IsCompressed(mediatype.GZipCompression) {
		gzipReader, err := gzip.NewReader(blobReader)
		if err != nil {
			if err == gzip.ErrHeader {
				return nil, errors.New("expected a gzip compressed tar")
			}
			return nil, err
		}
		blobReader = gzipReader
		defer errors.PropagateError(&rerr, gzipReader.Close)
	}

	fs := memoryfs.New()
	if err := fs.Mkdir("/", os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("unable to create blueprint directory: %w", err)
	}
	if err := tar.ExtractTar(ctx, blobReader, fs, tar.ToPath("/"), tar.Overwrite(true)); err != nil {
		return nil, fmt.Errorf("unable to extract blueprint from blob: %w", err)
	}

	bp, err := blueprint.BuildBlueprintFromPath(fs, "/")
	if err != nil {
		return nil, err
	}

	return &model.TypedResourceContent{
		Type:     mediatype.BlueprintType,
		Resource: bp,
	}, nil
}
