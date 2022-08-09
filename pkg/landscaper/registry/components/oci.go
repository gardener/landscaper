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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/config"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
)

// NewOCIRegistry creates a new oci registry from a oci config.
func NewOCIRegistry(log logging.Logger, config *config.OCIConfiguration, cache cache.Cache, predefinedComponentDescriptors ...*cdv2.ComponentDescriptor) (TypedRegistry, error) {
	client, err := ociclient.NewClient(log.Logr(), utils.WithConfiguration(config), ociclient.WithCache(cache))
	if err != nil {
		return nil, err
	}

	return NewOCIRegistryWithOCIClient(log, client, predefinedComponentDescriptors...)
}

// NewOCIRegistryWithOCIClient creates a new oci registry with a oci ociClient
// If supplied, it parses and stores predefined component descriptors.
func NewOCIRegistryWithOCIClient(log logging.Logger, client ociclient.Client, predefinedComponentDescriptors ...*cdv2.ComponentDescriptor) (TypedRegistry, error) {
	cache, err := newPredefinedComponentCache(predefinedComponentDescriptors...)
	if err != nil {
		return nil, err
	}
	res := cdoci.NewResolver(client).WithCache(cache).WithLog(log.Logr())
	return &ociClient{
		cache:     cache,
		ociClient: client,
		Resolver:  *res,
	}, nil
}

// ociClient wraps the central oci resolver and adds the supported resolve types.
type ociClient struct {
	cache     *predefinedComponentCache
	ociClient ociclient.Client
	cdoci.Resolver
}

// Type return the oci registry type that can be handled by this ociClient
func (r *ociClient) Type() string {
	return cdv2.OCIRegistryType
}

// predefinedComponentCache describes the cache for predefined components.
type predefinedComponentCache struct {
	components []*cdv2.ComponentDescriptor
}

var _ cdoci.Cache = &predefinedComponentCache{}

func newPredefinedComponentCache(predefinedComponentDescriptors ...*cdv2.ComponentDescriptor) (*predefinedComponentCache, error) {
	pcc := &predefinedComponentCache{
		components: make([]*cdv2.ComponentDescriptor, 0),
	}
	for _, cd := range predefinedComponentDescriptors {
		if cd == nil {
			continue
		}
		pcc.components = append(pcc.components, cd)

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
		err := extractCdFromLabel(&pcc.components, cd)
		if err != nil {
			return nil, err
		}
	}
	return pcc, nil
}

func (p predefinedComponentCache) Get(_ context.Context, repoCtx cdv2.OCIRegistryRepository, name, version string) (*cdv2.ComponentDescriptor, error) {
	for _, cd := range p.components {
		if cdv2.TypedObjectEqual(&repoCtx, cd.GetEffectiveRepositoryContext()) && name == cd.Name && version == cd.Version {
			return cd.DeepCopy(), nil
		}
	}
	return nil, ctf.NotFoundError
}

func (p predefinedComponentCache) Store(_ context.Context, _ *cdv2.ComponentDescriptor) error {
	// noop as we currently do not cache anything
	return nil
}

// resolveFromPredefined returns the respective component descriptor, if it exists within the cache
func (r *ociClient) resolveFromPredefined(repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver) {
	for _, cd := range r.cache.components {
		if repoCtx == cd.GetEffectiveRepositoryContext() && name == cd.Name && version == cd.Version {
			//TODO: add a resolver for local blobs
			return cd, &BlueprintResolver{ociClient: r.ociClient}
		}
	}
	return nil, nil
}

// ResolveWithBlobResolver wraps the component-spec component resolver by mocking the blobresolver for cached component descriptors.
// It also injects a blueprint oci artifact specific resolver.
func (r *ociClient) ResolveWithBlobResolver(ctx context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	//return cached component descriptor, if available
	if cd, blobResolver := r.resolveFromPredefined(repoCtx, name, version); cd != nil {
		return cd, blobResolver, nil
	}
	// resolve remote component descriptor
	cd, blobResolver, err := r.Resolver.ResolveWithBlobResolver(ctx, repoCtx, name, version)
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
