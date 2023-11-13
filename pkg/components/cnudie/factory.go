// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/cnudie/componentresolvers"
	"github.com/gardener/landscaper/pkg/components/cnudie/helmoci"
	"github.com/gardener/landscaper/pkg/components/cnudie/helmrepo"
	cnudieutils "github.com/gardener/landscaper/pkg/components/cnudie/utils"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type Factory struct{}

var _ model.Factory = &Factory{}

func (*Factory) NewRegistryAccess(ctx context.Context,
	fs vfs.FileSystem,
	secrets []corev1.Secret,
	sharedCache cache.Cache,
	localRegistryConfig *config.LocalRegistryConfiguration,
	ociRegistryConfig *config.OCIConfiguration,
	inlineCd *types.ComponentDescriptor,
	additionalBlobResolvers ...ctf.TypedBlobResolver) (model.RegistryAccess, error) {

	logger, _ := logging.FromContextOrNew(ctx, nil)

	if fs == nil {
		fs = osfs.New()
	}

	compResolver, err := componentresolvers.New(sharedCache)
	if err != nil {
		return nil, fmt.Errorf("unable to create component registry manager: %w", err)
	}

	if localRegistryConfig != nil {
		localRegistry, err := componentresolvers.NewLocalClient(fs, localRegistryConfig.RootPath)
		if err != nil {
			return nil, err
		}
		if err := compResolver.Set(localRegistry); err != nil {
			return nil, err
		}
		if err := compResolver.SetRegistryForAlias(localRegistry, componentresolvers.ComponentArchiveRepositoryType); err != nil {
			return nil, err
		}
	}

	// always add an oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if ociRegistryConfig != nil {
		ociConfigFiles = ociRegistryConfig.ConfigFiles
	}
	ociKeyring, err := credentials.NewBuilder(logger.Logr()).DisableDefaultConfig().
		WithFS(fs).
		FromConfigFiles(ociConfigFiles...).
		FromPullSecrets(secrets...).
		Build()

	if err != nil {
		return nil, err
	}

	ociClient, err := ociclient.NewClient(logger.Logr(),
		cnudieutils.WithConfiguration(ociRegistryConfig),
		ociclient.WithKeyring(ociKeyring),
		ociclient.WithCache(sharedCache),
	)
	if err != nil {
		return nil, err
	}

	componentsOCIRegistry, err := componentresolvers.NewOCIRegistryWithOCIClient(logger, ociClient, inlineCd)
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

// NewHelmRepoResource returns a helm chart resource that is stored in a helm chart repository.
func (*Factory) NewHelmRepoResource(ctx context.Context,
	helmChartRepo *helmv1alpha1.HelmChartRepo,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context) (model.TypedResourceProvider, error) {

	helmChartRepoResolver, err := helmrepo.NewBlobResolverForHelmRepo(ctx, lsClient, contextObj)
	if err != nil {
		return nil, err
	}

	resourceData, err := helmrepo.NewResourceDataForHelmRepo(helmChartRepo)
	if err != nil {
		return nil, err
	}

	return NewResource(resourceData, helmChartRepoResolver), nil
}

// NewHelmOCIResource returns a helm chart resource that is stored in an OCI registry.
func (*Factory) NewHelmOCIResource(ctx context.Context,
	fs vfs.FileSystem,
	ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (model.TypedResourceProvider, error) {

	blobResolver, err := helmoci.NewBlobResolverForHelmOCI(ctx, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, err
	}

	resourceData, err := helmoci.NewResourceDataForHelmOCI(ociImageRef)
	if err != nil {
		return nil, err
	}

	return NewResource(resourceData, blobResolver), nil
}
