// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver

import (
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

type HelmChartResolver struct {
	ociClient ociclient.Client
}

// NewHelmResolver returns a new helm chart resolver.
// It implements the blob resolver interface, so it can resolve component descriptor defined resources.
func NewHelmResolver(client ociclient.Client) ctf.TypedBlobResolver {
	return HelmChartResolver{
		ociClient: client,
	}
}

func (h HelmChartResolver) CanResolve(res cdv2.Resource) bool {
	if res.GetType() != HelmChartResourceType && res.GetType() != OldHelmResourceType {
		return false
	}
	return res.Access != nil && res.Access.GetType() == cdv2.OCIRegistryType
}

func (h HelmChartResolver) Info(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error) {
	return h.resolve(ctx, res, nil)
}

func (h HelmChartResolver) Resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	return h.resolve(ctx, res, writer)
}

func (h HelmChartResolver) resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	ociArtifactAccess := &cdv2.OCIRegistryAccess{}
	if err := cdv2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, ociArtifactAccess); err != nil {
		return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	manifest, err := h.ociClient.GetManifest(ctx, ociArtifactAccess.ImageReference)
	if err != nil {
		return nil, err
	}

	// since helm 3.7 two layers are valid.
	// one containing the chart and one containing the provenance files.
	if len(manifest.Layers) > 2 || len(manifest.Layers) == 0 {
		return nil, fmt.Errorf("expected one or two layers but found %d", len(manifest.Layers))
	}

	if chartLayers := ociclient.GetLayerByMediaType(manifest.Layers, ChartLayerMediaType); len(chartLayers) != 0 {
		if manifest.Config.MediaType != HelmChartConfigMediaType {
			return nil, fmt.Errorf("unexpected media type of helm config. Expected %s but got %s", HelmChartConfigMediaType, manifest.Config.MediaType)
		}
		chartLayer := chartLayers[0]
		// todo: check verify the signature if a provenance layer is present.
		if writer != nil {
			if err := h.ociClient.Fetch(ctx, ociArtifactAccess.ImageReference, chartLayer, writer); err != nil {
				return nil, err
			}
		}
		return &ctf.BlobInfo{
			MediaType: ChartLayerMediaType,
			Digest:    chartLayer.Digest.String(),
			Size:      chartLayer.Size,
		}, nil
	}
	if legacyChartLayers := ociclient.GetLayerByMediaType(manifest.Layers, LegacyChartLayerMediaType); len(legacyChartLayers) != 0 {
		logging.FromContextOrDiscard(ctx).Info("LEGACY Helm Chart used", "ref", ociArtifactAccess.ImageReference)
		chartLayer := legacyChartLayers[0]
		if writer != nil {
			if err := h.ociClient.Fetch(ctx, ociArtifactAccess.ImageReference, chartLayer, writer); err != nil {
				return nil, err
			}
		}
		return &ctf.BlobInfo{
			MediaType: LegacyChartLayerMediaType,
			Digest:    chartLayer.Digest.String(),
			Size:      chartLayer.Size,
		}, nil
	}

	return nil, fmt.Errorf("unknown oci artifact of type %s", manifest.Config.MediaType)
}
