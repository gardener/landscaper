// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry

import (
	"context"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/apis/mediatype"

	"github.com/gardener/component-cli/ociclient"
)

// BlueprintResolver is a blob resolver that can resolve
// blueprints that are defined as their independent oci artifact.
type BlueprintResolver struct {
	ociClient ociclient.Client
}

var _ ctf.TypedBlobResolver = &BlueprintResolver{}

func (b BlueprintResolver) CanResolve(res cdv2.Resource) bool {
	if res.GetType() != mediatype.BlueprintType && res.GetType() != mediatype.OldBlueprintType {
		return false
	}
	return res.Access != nil && res.Access.GetType() == cdv2.OCIRegistryType
}

func (b *BlueprintResolver) Info(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error) {
	return b.resolve(ctx, res, nil)
}

func (b *BlueprintResolver) Resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	return b.resolve(ctx, res, writer)
}

func (b *BlueprintResolver) resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	ociArtifactAccess := &cdv2.OCIRegistryAccess{}
	if err := cdv2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, ociArtifactAccess); err != nil {
		return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	manifest, err := b.ociClient.GetManifest(ctx, ociArtifactAccess.ImageReference)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch manifest for oci artifact %s: %w", ociArtifactAccess.ImageReference, err)
	}
	if len(manifest.Layers) != 1 {
		return nil, fmt.Errorf("expected blueprint oci artifacts to habe exactly o1 layer but got %d", len(manifest.Layers))
	}
	blueprintLayer := manifest.Layers[0]

	if writer != nil {
		if err := b.ociClient.Fetch(ctx, ociArtifactAccess.ImageReference, blueprintLayer, writer); err != nil {
			return nil, err
		}
	}

	return &ctf.BlobInfo{
		MediaType: blueprintLayer.MediaType,
		Digest:    blueprintLayer.Digest.String(),
		Size:      blueprintLayer.Size,
	}, nil
}
