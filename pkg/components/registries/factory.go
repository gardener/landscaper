package registries

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
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

// Factory implements the model.Factory interface.
// Its purpose is to select either the cnudie or ocm implementation of the same interface.
type Factory struct {
	cnudieFactory *cnudie.Factory
	// ocmFactory *cnudie.RegistryAccessBuilder
}

var _ model.Factory = &Factory{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) NewRegistryAccess(ctx context.Context,
	secrets []corev1.Secret,
	sharedCache cache.Cache,
	localRegistryConfig *config.LocalRegistryConfiguration,
	ociRegistryConfig *config.OCIConfiguration,
	inlineCd *types.ComponentDescriptor,
	additionalBlobResolvers ...ctf.TypedBlobResolver) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewRegistryAccess(ctx, secrets, sharedCache, localRegistryConfig, ociRegistryConfig, inlineCd, additionalBlobResolvers...)
}

func (f *Factory) NewRegistryAccessFromOciOptions(ctx context.Context,
	log logr.Logger,
	fs vfs.FileSystem,
	allowPlainHttp bool,
	skipTLSVerify bool,
	registryConfigPath string,
	concourseConfigPath string,
	predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewRegistryAccessFromOciOptions(ctx, log, fs, allowPlainHttp, skipTLSVerify, registryConfigPath, concourseConfigPath, predefinedComponentDescriptors...)
}

func (f *Factory) NewRegistryAccessForHelm(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache,
	ref *helmv1alpha1.RemoteChartReference) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewRegistryAccessForHelm(ctx, lsClient, contextObj, registryPullSecrets, ociConfig, sharedCache, ref)
}

func (f *Factory) NewOCIRegistryAccess(ctx context.Context,
	config *config.OCIConfiguration,
	cache cache.Cache,
	predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewOCIRegistryAccess(ctx, config, cache, predefinedComponentDescriptors...)
}

func (f *Factory) NewOCIRegistryAccessFromDockerAuthConfig(ctx context.Context,
	fs vfs.FileSystem,
	registrySecretBasePath string,
	predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewOCIRegistryAccessFromDockerAuthConfig(ctx, fs, registrySecretBasePath, predefinedComponentDescriptors...)
}

func (f *Factory) NewOCITestRegistryAccess(address, username, password string) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewOCITestRegistryAccess(address, username, password)
}

func (f *Factory) NewLocalRegistryAccess(rootPath string) (model.RegistryAccess, error) {
	return f.cnudieFactory.NewLocalRegistryAccess(rootPath)
}

// NewHelmRepoResource returns a helm chart resource that is stored in a helm chart repository.
func (f *Factory) NewHelmRepoResource(ctx context.Context,
	helmChartRepo *helmv1alpha1.HelmChartRepo,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context) (model.Resource, error) {
	return f.cnudieFactory.NewHelmRepoResource(ctx, helmChartRepo, lsClient, contextObj)
}

func (f *Factory) NewHelmOCIResource(ctx context.Context,
	ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (model.Resource, error) {
	return f.cnudieFactory.NewHelmOCIResource(ctx, ociImageRef, registryPullSecrets, ociConfig, sharedCache)
}
