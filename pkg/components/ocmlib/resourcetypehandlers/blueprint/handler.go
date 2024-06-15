// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package blueprint

import (
	"context"
	"fmt"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	bpdownload "github.com/open-component-model/ocm/pkg/contexts/ocm/download/handlers/blueprint"

	"github.com/gardener/landscaper/apis/mediatype"
	componentscommon "github.com/gardener/landscaper/pkg/components/common"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmlib/registries"
)

func init() {
	registries.Registry.Register(mediatype.BlueprintType, New())
	registries.Registry.Register(mediatype.OldBlueprintType, New())
}

type BlueprintHandler struct{}

func New() *BlueprintHandler {
	return &BlueprintHandler{}
}

func (h *BlueprintHandler) GetResourceContent(ctx context.Context, _ model.Resource, access ocm.ResourceAccess) (*model.TypedResourceContent, error) {
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

	return typedResourceContent, nil
}

func (h *BlueprintHandler) Prepare(ctx context.Context, fs vfs.FileSystem) (*model.TypedResourceContent, error) {
	bp, err := componentscommon.BuildBlueprintFromPath(fs, "/")
	if err != nil {
		return nil, err
	}

	return &model.TypedResourceContent{
		Type:     mediatype.BlueprintType,
		Resource: bp,
	}, nil
}
