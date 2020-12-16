// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/pkg/apis/config"
	confighelper "github.com/gardener/landscaper/pkg/apis/config/helper"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
)

// ociClient is a component descriptor repository implementation
// that resolves component references stored in an oci repository.
type ociClient struct {
	ociClient ociclient.Client
	resolver  *cdoci.Resolver
}

// NewOCIRegistry creates a new oci registry from a oci config.
func NewOCIRegistry(log logr.Logger, config *config.OCIConfiguration) (TypedRegistry, error) {
	client, err := ociclient.NewClient(log, confighelper.WithConfiguration(config))
	if err != nil {
		return nil, err
	}

	return &ociClient{
		ociClient: client,
		resolver:  cdoci.NewResolver().WithOCIClient(client),
	}, nil
}

// NewOCIRegistryWithOCIClient creates a new oci registry with a oci ociClient
func NewOCIRegistryWithOCIClient(log logr.Logger, client ociclient.Client) (TypedRegistry, error) {
	return &ociClient{
		ociClient: client,
		resolver:  cdoci.NewResolver().WithOCIClient(client),
	}, nil
}

// Type return the oci registry type that can be handled by this ociClient
func (r *ociClient) Type() string {
	return cdv2.OCIRegistryType
}

func (r *ociClient) InjectCache(c cache.Cache) error {
	return cache.InjectCacheInto(r.ociClient, c)
}

// Get resolves a reference and returns the component descriptor.
func (r *ociClient) Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	cd, blobResolver, err := r.resolver.WithRepositoryContext(repoCtx).Resolve(ctx, name, version)
	if err != nil {
		return nil, nil, err
	}
	// automatically add blueprint resolver
	aggBlobResolver, err := ctf.AggregateBlobResolvers(blobResolver, &BlueprintResolver{ociClient: r.ociClient})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to add blueprint resolver")
	}
	return cd, aggBlobResolver, nil
}
