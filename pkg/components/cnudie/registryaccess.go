// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/osfs"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type RegistryAccess struct {
	componentResolver       ctf.ComponentResolver
	additionalBlobResolvers []ctf.TypedBlobResolver
}

var _ model.RegistryAccess = &RegistryAccess{}

func NewCnudieRegistry(ctx context.Context,
	secrets []corev1.Secret,
	sharedCache cache.Cache,
	localRegistryConfig *config.LocalRegistryConfiguration,
	ociRegistryConfig *config.OCIConfiguration,
	inlineCd *cdv2.ComponentDescriptor,
	additionalBlobResolvers ...ctf.TypedBlobResolver) (model.RegistryAccess, error) {

	logger, _ := logging.FromContextOrNew(ctx, nil)

	compResolver, err := componentsregistry.New(sharedCache)
	if err != nil {
		return nil, fmt.Errorf("unable to create component registry manager: %w", err)
	}
	if localRegistryConfig != nil {
		componentsOCIRegistry, err := componentsregistry.NewLocalClient(localRegistryConfig.RootPath)
		if err != nil {
			return nil, err
		}
		if err := compResolver.Set(componentsOCIRegistry); err != nil {
			return nil, err
		}
	}

	// always add an oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if ociRegistryConfig != nil {
		ociConfigFiles = ociRegistryConfig.ConfigFiles
	}
	ociKeyring, err := credentials.NewBuilder(logger.Logr()).DisableDefaultConfig().
		WithFS(osfs.New()).
		FromConfigFiles(ociConfigFiles...).
		FromPullSecrets(secrets...).
		Build()
	if err != nil {
		return nil, err
	}

	ociClient, err := ociclient.NewClient(logger.Logr(),
		utils.WithConfiguration(ociRegistryConfig),
		ociclient.WithKeyring(ociKeyring),
		ociclient.WithCache(sharedCache),
	)
	if err != nil {
		return nil, err
	}

	componentsOCIRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(logger, ociClient, inlineCd)
	if err != nil {
		return nil, err
	}
	if err := compResolver.Set(componentsOCIRegistry); err != nil {
		return nil, err
	}

	return &RegistryAccess{
		componentResolver:       compResolver,
		additionalBlobResolvers: additionalBlobResolvers,
	}, nil
}

func NewLocalRegistryAccess(rootPath string) (model.RegistryAccess, error) {
	localComponentResolver, err := componentsregistry.NewLocalClient(rootPath)
	if err != nil {
		return nil, err
	}

	return &RegistryAccess{
		componentResolver: localComponentResolver,
	}, nil
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	cd, blobResolver, err := r.componentResolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}

	for i := range r.additionalBlobResolvers {
		additionalBlobResolver := r.additionalBlobResolvers[i]
		blobResolver, err = ctf.AggregateBlobResolvers(blobResolver, additionalBlobResolver)
		if err != nil {
			return nil, fmt.Errorf("unable to aggregate blob resolvers: %w", err)
		}
	}

	return newComponentVersion(r, cd, blobResolver), nil
}

// temporary
func (r *RegistryAccess) GetComponentResolver() ctf.ComponentResolver {
	return r.componentResolver
}
