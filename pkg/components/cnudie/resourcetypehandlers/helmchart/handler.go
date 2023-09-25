// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helmchart

import (
	"bytes"
	"context"
	"fmt"
	"io"

	chartloader "helm.sh/helm/v3/pkg/chart/loader"

	"github.com/gardener/landscaper/pkg/components/cnudie/registries"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

func init() {
	registries.Registry.Register(types.HelmChartResourceType, New())
	registries.Registry.Register(types.OldHelmResourceType, New())
}

type HelmChartHandler struct{}

func New() *HelmChartHandler {
	return &HelmChartHandler{}
}

func (h *HelmChartHandler) GetResourceContent(ctx context.Context, r model.Resource, blobResolver model.BlobResolver) (*model.TypedResourceContent, error) {
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

	return typedResourceContent, nil
}

func (h *HelmChartHandler) Prepare(ctx context.Context, data io.Reader, info *types.BlobInfo) (_ *model.TypedResourceContent, rerr error) {
	blobReader := data

	chart, err := chartloader.LoadArchive(blobReader)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}

	return &model.TypedResourceContent{
		Type:     types.HelmChartResourceType,
		Resource: chart,
	}, nil
}
