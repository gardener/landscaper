// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/config"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
)

// ociClient is a component descriptor repository implementation
// that resolves component references stored in an oci repository.
// It can also cache local component descriptors.
type ociClient struct {
	ociClient ociclient.Client
	resolver  *cdoci.Resolver
	cache     []*cdv2.ComponentDescriptor
}

// NewOCIRegistry creates a new oci registry from a oci config.
func NewOCIRegistry(log logr.Logger, config *config.OCIConfiguration, predefinedComponentDescriptors ...*cdv2.ComponentDescriptor) (TypedRegistry, error) {
	client, err := ociclient.NewClient(log, utils.WithConfiguration(config))
	if err != nil {
		return nil, err
	}

	return NewOCIRegistryWithOCIClient(client, predefinedComponentDescriptors...)
}

// NewOCIRegistryWithOCIClient creates a new oci registry with a oci ociClient
// If supplied, it parses and stores predefined component descriptors.
func NewOCIRegistryWithOCIClient(client ociclient.Client, predefinedComponentDescriptors ...*cdv2.ComponentDescriptor) (TypedRegistry, error) {
	cache := make([]*cdv2.ComponentDescriptor, 0)
	for _, cd := range predefinedComponentDescriptors {
		if cd == nil {
			continue
		}
		cache = append(cache, cd)

		var extractCdFromLabel func(*[]*cdv2.ComponentDescriptor, *cdv2.ComponentDescriptor) error
		extractCdFromLabel = func(cdList *[]*cdv2.ComponentDescriptor, cd1 *cdv2.ComponentDescriptor) error {
			for _, cdRef := range cd1.ComponentReferences {
				if label, exists := cdRef.Labels.Get(lsv1alpha1.InlineComponentDescriptorLabel); exists {
					var cdFromLabel cdv2.ComponentDescriptor
					if err := codec.Decode(label, &cdFromLabel); err != nil {
						return err
					}
					*cdList = append(*cdList, &cdFromLabel)
					err := extractCdFromLabel(cdList, &cdFromLabel)
					return err
				}
			}
			return nil
		}
		err := extractCdFromLabel(&cache, cd)
		if err != nil {
			return nil, err
		}
	}

	return &ociClient{
		ociClient: client,
		resolver:  cdoci.NewResolver().WithOCIClient(client),
		cache:     cache,
	}, nil
}

// Type return the oci registry type that can be handled by this ociClient
func (r *ociClient) Type() string {
	return cdv2.OCIRegistryType
}

func (r *ociClient) InjectCache(c cache.Cache) error {
	return cache.InjectCacheInto(r.ociClient, c)
}

// resolveFromPredefined returns the respective component descriptor, if it exists within the cache
func (r *ociClient) resolveFromPredefined(repoCtx cdv2.RepositoryContext, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver) {
	for _, cd := range r.cache {
		if repoCtx == cd.GetEffectiveRepositoryContext() && name == cd.Name && version == cd.Version {
			//TODO: add a resolver for local blobs
			return cd, &BlueprintResolver{ociClient: r.ociClient}
		}
	}
	return nil, nil
}

// Resolve resolves a reference and returns the component descriptor.
// Predefined inline component descriptors take precedence.
func (r *ociClient) Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	//return cached component descriptor, if availalbe
	if cd, blobResolver := r.resolveFromPredefined(repoCtx, name, version); cd != nil {
		return cd, blobResolver, nil
	}
	// resolve remote component descriptor
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
