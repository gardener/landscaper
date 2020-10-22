// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactsregistry

import (
	"context"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type ociRegistry struct {
	oci oci.Client
}

var _ cache.InjectCache = &ociRegistry{}

// NewOCIRegistry creates a new oci ociRegistry from a oci config.
func NewOCIRegistry(log logr.Logger, config *config.OCIConfiguration) (TypedRegistry, error) {
	client, err := oci.NewClient(log, oci.WithConfiguration(config))
	if err != nil {
		return nil, err
	}

	return &ociRegistry{
		oci: client,
	}, nil
}

// NewOCIRegistryWithOCIClient creates a new oci ociRegistry with a oci client
func NewOCIRegistryWithOCIClient(log logr.Logger, client oci.Client) (TypedRegistry, error) {
	return &ociRegistry{
		oci: client,
	}, nil
}

func (r *ociRegistry) Type() string {
	return cdv2.OCIRegistryType
}

func (r *ociRegistry) InjectCache(c cache.Cache) error {
	return cache.InjectCacheInto(r.oci, c)
}

func (r *ociRegistry) GetBlob(ctx context.Context, access cdv2.TypedObjectAccessor, writer io.Writer) (string, error) {
	if access.GetType() != cdv2.OCIRegistryType {
		return "", fmt.Errorf("wrong access type '%s' expected '%s'", access.GetType(), cdv2.OCIRegistryType)
	}
	ociComp := access.(*cdv2.OCIRegistryAccess)
	ociRef := ociComp.ImageReference

	manifest, err := r.oci.GetManifest(ctx, ociRef)
	if err != nil {
		return "", err
	}
	if len(manifest.Layers) == 0 {
		return "", fmt.Errorf("no layer defined")
	}
	blobLayer := manifest.Layers[0]

	if err := r.oci.Fetch(ctx, ociRef, blobLayer, writer); err != nil {
		return "", err
	}
	return blobLayer.MediaType, nil
}
