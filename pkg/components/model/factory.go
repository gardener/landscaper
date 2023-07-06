package model

import (
	"context"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type Factory interface {
	NewRegistryAccess(ctx context.Context,
		secrets []corev1.Secret,
		sharedCache cache.Cache,
		localRegistryConfig *config.LocalRegistryConfiguration,
		ociRegistryConfig *config.OCIConfiguration,
		inlineCd *types.ComponentDescriptor,
		additionalBlobResolvers ...ctf.TypedBlobResolver) (RegistryAccess, error)

	NewRegistryAccessFromOciOptions(ctx context.Context,
		log logr.Logger,
		fs vfs.FileSystem,
		allowPlainHttp bool,
		skipTLSVerify bool,
		registryConfigPath string,
		concourseConfigPath string,
		predefinedComponentDescriptors ...*types.ComponentDescriptor) (RegistryAccess, error)

	NewRegistryAccessForHelm(ctx context.Context,
		lsClient client.Client,
		contextObj *lsv1alpha1.Context,
		registryPullSecrets []corev1.Secret,
		ociConfig *config.OCIConfiguration,
		sharedCache cache.Cache,
		ref *helmv1alpha1.RemoteChartReference) (RegistryAccess, error)

	NewOCIRegistryAccess(ctx context.Context,
		config *config.OCIConfiguration,
		cache cache.Cache,
		predefinedComponentDescriptors ...*types.ComponentDescriptor) (RegistryAccess, error)

	NewOCIRegistryAccessFromDockerAuthConfig(ctx context.Context,
		fs vfs.FileSystem,
		registrySecretBasePath string,
		predefinedComponentDescriptors ...*types.ComponentDescriptor) (RegistryAccess, error)

	NewOCITestRegistryAccess(address, username, password string) (RegistryAccess, error)

	// NewLocalRegistryAccess returns a RegistryAccess whose component descriptors are stored in the local file system
	// below the given root path (and its subdirectories), in files with name component-descriptor.yaml
	// This constructor is intended to create test objects.
	NewLocalRegistryAccess(rootPath string) (RegistryAccess, error)

	// NewHelmRepoResource returns a helm chart resource that is stored in a helm chart repository.
	NewHelmRepoResource(ctx context.Context,
		helmChartRepo *helmv1alpha1.HelmChartRepo,
		lsClient client.Client,
		contextObj *lsv1alpha1.Context) (Resource, error)

	// NewHelmOCIResource returns a helm chart resource that is stored in an OCI registry.
	NewHelmOCIResource(ctx context.Context,
		ociImageRef string,
		registryPullSecrets []corev1.Secret,
		ociConfig *config.OCIConfiguration,
		sharedCache cache.Cache) (Resource, error)
}
