// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
)

type HelmChartResolver struct {
	oci ociclient.Client
}

// Returns a new helm chart resolver.
// It implements the blob resolver interface so it can resolve component descriptor defined resources.
func NewHelmResolver(client ociclient.Client) ctf.TypedBlobResolver {
	return HelmChartResolver{
		oci: client,
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

	manifest, err := h.oci.GetManifest(ctx, ociArtifactAccess.ImageReference)
	if err != nil {
		return nil, err
	}

	if len(manifest.Layers) != 1 {
		return nil, errors.New("unexpected number of layers")
	}

	if manifest.Layers[0].MediaType != HelmChartContentLayerMediaType {
		return nil, errors.New("unexpected media type of content")
	}

	if writer != nil {
		if err := h.oci.Fetch(ctx, ociArtifactAccess.ImageReference, manifest.Layers[0], writer); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
