// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

type ociArtifactDownloader struct {
	client ociclient.Client
	cache  cache.Cache
}

// NewOCIArtifactDownloader creates a new ociArtifactDownloader
func NewOCIArtifactDownloader(client ociclient.Client, cache cache.Cache) (process.ResourceStreamProcessor, error) {
	if client == nil {
		return nil, errors.New("client must not be nil")
	}

	if cache == nil {
		return nil, errors.New("cache must not be nil")
	}

	obj := ociArtifactDownloader{
		client: client,
		cache:  cache,
	}
	return &obj, nil
}

func (d *ociArtifactDownloader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, _, err := utils.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read processor message: %w", err)
	}

	if res.Access.GetType() != cdv2.OCIRegistryType {
		return fmt.Errorf("unsupported access type: %s", res.Access.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return fmt.Errorf("unable to decode resource access: %w", err)
	}

	ociArtifact, err := d.client.GetOCIArtifact(ctx, ociAccess.ImageReference)
	if err != nil {
		return fmt.Errorf("unable to get oci artifact: %w", err)
	}

	if ociArtifact.IsManifest() {
		if err := d.fetchConfigAndLayerBlobs(ctx, ociAccess.ImageReference, ociArtifact.GetManifest().Data); err != nil {
			return err
		}
	} else if ociArtifact.IsIndex() {
		for _, m := range ociArtifact.GetIndex().Manifests {
			if err := d.fetchConfigAndLayerBlobs(ctx, ociAccess.ImageReference, m.Data); err != nil {
				return err
			}
		}
	}

	blobReader, err := utils.SerializeOCIArtifact(*ociArtifact, d.cache)
	if err != nil {
		return fmt.Errorf("unable to serialize oci artifact: %w", err)
	}
	defer blobReader.Close()

	if err := utils.WriteProcessorMessage(*cd, res, blobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}

func (d *ociArtifactDownloader) fetchConfigAndLayerBlobs(ctx context.Context, ref string, manifest *ocispecv1.Manifest) error {
	buf := bytes.NewBuffer([]byte{})
	if err := d.client.Fetch(ctx, ref, manifest.Config, buf); err != nil {
		return fmt.Errorf("unable to fetch config blob: %w", err)
	}
	for _, l := range manifest.Layers {
		buf := bytes.NewBuffer([]byte{})
		if err := d.client.Fetch(ctx, ref, l, buf); err != nil {
			return fmt.Errorf("unable to fetch layer blob: %w", err)
		}
	}
	return nil
}
