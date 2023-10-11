// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package blueprint

import (
	"context"
	"fmt"

	"github.com/open-component-model/ocm/pkg/common"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	bpdownload "github.com/open-component-model/ocm/pkg/contexts/ocm/download/handlers/blueprint"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmlib/registries"
)

func init() {
	registries.Registry.Register(mediatype.BlueprintType, New())
	registries.Registry.Register(mediatype.OldBlueprintType, New())
}

type BlueprintHandler struct {
	cache *blueprint.Store
}

func New() *BlueprintHandler {
	return &BlueprintHandler{
		cache: blueprint.GetBlueprintStore(),
	}
}
func (h *BlueprintHandler) GetResourceContent(ctx context.Context, r model.Resource, access ocm.ResourceAccess) (*model.TypedResourceContent, error) {
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
	pr := common.NewPrinter(nil)
	ok, _, err := bpdownload.New().Download(pr, access, filepath.Join("/"), fs)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("artifact does not match blueprint downloader (check config media type)")
	}

	typedResourceContent, err := h.Prepare(ctx, fs)
	if err != nil {
		return nil, err
	}

	_, err = blueprint.GetBlueprintStore().Put(ctx, r.GetCachingIdentity(ctx), typedResourceContent)
	if err != nil {
		return nil, err
	}

	return typedResourceContent, nil
}

func (h *BlueprintHandler) Prepare(ctx context.Context, fs vfs.FileSystem) (*model.TypedResourceContent, error) {
	bp, err := blueprint.BuildBlueprintFromPath(fs, "/")
	if err != nil {
		return nil, err
	}

	return &model.TypedResourceContent{
		Type:     mediatype.BlueprintType,
		Resource: bp,
	}, nil
}
