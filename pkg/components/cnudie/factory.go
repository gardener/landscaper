package cnudie

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	testcred "github.com/gardener/component-cli/ociclient/credentials"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
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
	secrets []corev1.Secret,
	sharedCache cache.Cache,
	localRegistryConfig *config.LocalRegistryConfiguration,
	ociRegistryConfig *config.OCIConfiguration,
	inlineCd *types.ComponentDescriptor,
	additionalBlobResolvers ...ctf.TypedBlobResolver) (model.RegistryAccess, error) {

	logger, _ := logging.FromContextOrNew(ctx, nil)

	compResolver, err := componentresolvers.New(sharedCache)
	if err != nil {
		return nil, fmt.Errorf("unable to create component registry manager: %w", err)
	}

	if localRegistryConfig != nil {
		localRegistry, err := componentresolvers.NewLocalClient(localRegistryConfig.RootPath)
		if err != nil {
			return nil, err
		}
		if err := compResolver.Set(localRegistry); err != nil {
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

func (f *Factory) NewRegistryAccessFromOciOptions(ctx context.Context,
	log logr.Logger,
	fs vfs.FileSystem,
	allowPlainHttp bool,
	skipTLSVerify bool,
	registryConfigPath string,
	concourseConfigPath string,
	predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {

	ociOptions := ociopts.Options{
		AllowPlainHttp:      allowPlainHttp,
		SkipTLSVerify:       skipTLSVerify,
		RegistryConfigPath:  registryConfigPath,
		ConcourseConfigPath: concourseConfigPath,
	}

	ociClient, _, err := ociOptions.Build(log, fs)
	if err != nil {
		return nil, fmt.Errorf("unable to build oci client: %w", err)
	}

	//compResolver := cdoci.NewResolver(ociClient)
	compResolver, err := componentresolvers.NewOCIRegistryWithOCIClient(logging.Wrap(log), ociClient, predefinedComponentDescriptors...)
	if err != nil {
		return nil, fmt.Errorf("unable to build component resolver with oci client: %w", err)
	}

	return &RegistryAccess{
		componentResolver: compResolver,
	}, nil
}

func (f *Factory) NewRegistryAccessForHelm(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache,
	ref *helmv1alpha1.RemoteChartReference) (model.RegistryAccess, error) {

	helmChartOCIResolver, err := helmoci.NewBlobResolverForHelmOCI(ctx, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, err
	}

	helmChartRepoResolver, err := helmrepo.NewBlobResolverForHelmRepo(ctx, lsClient, contextObj)
	if err != nil {
		return nil, err
	}

	registryAccess, err := f.NewRegistryAccess(ctx, registryPullSecrets, sharedCache, nil, ociConfig, ref.Inline,
		helmChartOCIResolver, helmChartRepoResolver)
	if err != nil {
		return nil, fmt.Errorf("unable to build registry access for helm charts: %w", err)
	}

	return registryAccess, nil
}

func (*Factory) NewOCIRegistryAccess(ctx context.Context,
	config *config.OCIConfiguration,
	cache cache.Cache,
	predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {

	log, _ := logging.FromContextOrNew(ctx, nil)

	ociClient, err := ociclient.NewClient(log.Logr(), cnudieutils.WithConfiguration(config), ociclient.WithCache(cache))
	if err != nil {
		return nil, err
	}

	componentResolver, err := componentresolvers.NewOCIRegistryWithOCIClient(log, ociClient, predefinedComponentDescriptors...)
	if err != nil {
		return nil, err
	}

	return &RegistryAccess{
		componentResolver:       componentResolver,
		additionalBlobResolvers: nil,
	}, nil
}

func (*Factory) NewOCIRegistryAccessFromDockerAuthConfig(ctx context.Context,
	fs vfs.FileSystem,
	registrySecretBasePath string,
	predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {

	log, ctx := logging.FromContextOrNew(ctx, nil)

	ociClient, err := createOciClientFromDockerAuthConfig(ctx, fs, registrySecretBasePath)
	if err != nil {
		return nil, err
	}

	componentResolver, err := componentresolvers.NewOCIRegistryWithOCIClient(log, ociClient, predefinedComponentDescriptors...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to setup components registry")
	}

	return &RegistryAccess{
		componentResolver:       componentResolver,
		additionalBlobResolvers: nil,
	}, nil
}

func createOciClientFromDockerAuthConfig(ctx context.Context, fs vfs.FileSystem, registryPullSecretsDir string) (ociclient.Client, error) {
	log, _ := logging.FromContextOrNew(ctx, nil)
	var secrets []string
	err := vfs.Walk(fs, registryPullSecretsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != corev1.DockerConfigJsonKey {
			return nil
		}

		secrets = append(secrets, path)

		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to add local registry pull secrets: %w", err)
	}

	keyring, err := credentials.CreateOCIRegistryKeyringFromFilesystem(nil, secrets, fs)
	if err != nil {
		return nil, err
	}

	ociClient, err := ociclient.NewClient(log.Logr(), ociclient.WithKeyring(keyring))
	if err != nil {
		return nil, err
	}

	return ociClient, err
}

func (*Factory) NewOCITestRegistryAccess(address, username, password string) (model.RegistryAccess, error) {
	keyring := testcred.New()
	if err := keyring.AddAuthConfig(address, testcred.AuthConfig{Username: username, Password: password}); err != nil {
		return nil, err
	}

	ociCache, err := cache.NewCache(logging.Discard().Logr())
	if err != nil {
		return nil, err
	}

	ociClient, err := ociclient.NewClient(logging.Discard().Logr(), ociclient.WithKeyring(keyring), ociclient.WithCache(ociCache))
	if err != nil {
		return nil, err
	}

	return &RegistryAccess{
		componentResolver:       cdoci.NewResolver(ociClient),
		additionalBlobResolvers: nil,
	}, nil
}

func (*Factory) NewLocalRegistryAccess(rootPath string) (model.RegistryAccess, error) {
	localComponentResolver, err := componentresolvers.NewLocalClient(rootPath)
	if err != nil {
		return nil, err
	}

	return &RegistryAccess{
		componentResolver: localComponentResolver,
	}, nil
}

// NewHelmRepoResource returns a helm chart resource that is stored in a helm chart repository.
func (*Factory) NewHelmRepoResource(ctx context.Context,
	helmChartRepo *helmv1alpha1.HelmChartRepo,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context) (model.Resource, error) {

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
	ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (model.Resource, error) {

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
