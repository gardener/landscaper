package model

import (
	"context"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-spec/bindings-go/ctf"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type Factory interface {
	SetApplicationLogger(logger logging.Logger)
	NewRegistryAccess(ctx context.Context,
		fs vfs.FileSystem,
		secrets []corev1.Secret,
		sharedCache cache.Cache,
		localRegistryConfig *config.LocalRegistryConfiguration,
		ociRegistryConfig *config.OCIConfiguration,
		inlineCd *types.ComponentDescriptor,
		additionalBlobResolvers ...ctf.TypedBlobResolver) (RegistryAccess, error)

	// NewHelmRepoResource returns a helm chart resource that is stored in a helm chart repository.
	NewHelmRepoResource(ctx context.Context,
		helmChartRepo *helmv1alpha1.HelmChartRepo,
		lsClient client.Client,
		contextObj *lsv1alpha1.Context) (TypedResourceProvider, error)

	// NewHelmOCIResource returns a helm chart resource that is stored in an OCI registry.
	NewHelmOCIResource(ctx context.Context,
		ociImageRef string,
		registryPullSecrets []corev1.Secret,
		ociConfig *config.OCIConfiguration,
		sharedCache cache.Cache) (TypedResourceProvider, error)
}
