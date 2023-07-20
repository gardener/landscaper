package ocmfacade

import (
	"context"
	"fmt"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/inlinecompdesc"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/blobvfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/compvfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/localrootfs"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	//logger, _ := logging.FromContextOrNew(ctx, nil)
	registryAccess := &RegistryAccess{}
	registryAccess.octx = ocm.New(datacontext.MODE_EXTENDED)
	registryAccess.session = ocm.NewSession(datacontext.NewSession())

	//cpi.RegisterRepositoryType(cpi.NewRepositoryType[*comparch.RepositorySpec]("local", nil))

	ociConfigFiles := make([]string, 0)
	if ociRegistryConfig != nil {
		ociConfigFiles = ociRegistryConfig.ConfigFiles
	}

	if localRegistryConfig != nil {
		pfs, err := projectionfs.New(osfs.New(), localRegistryConfig.RootPath)
		if err != nil {
			return nil, err
		}
		localrootfs.Set(registryAccess.octx, pfs)
		compvfs.Set(registryAccess.octx, pfs)
		blobvfs.Set(registryAccess.octx, pfs)
	}

	if inlineCd != nil {
		cd, err := runtime.DefaultYAMLEncoding.Marshal(inlineCd)
		if err != nil {
			return nil, err
		}
		inline, err := inlinecompdesc.New(cd)
		if err != nil {
			return nil, err
		}
		descriptors, err := inline.GetFlatList()
		if err != nil {
			return nil, err
		}

		memfs := memoryfs.New()

		for i, cd := range descriptors {
			data, err := compdesc.Encode(cd)
			if err != nil {
				return nil, err
			}
			file, err := memfs.Create(fmt.Sprintf("component-descriptor%d.yaml", i))
			if err != nil {
				return nil, err
			}
			if _, err := file.Write(data); err != nil {
				return nil, err
			}
			if err := file.Close(); err != nil {
				return nil, err
			}
		}
		compvfs.Set(registryAccess.octx, memfs)
	}

	// set available default credentials from dockerconfig files
	var spec *dockerconfig.RepositorySpec
	for _, path := range ociConfigFiles {
		spec = dockerconfig.NewRepositorySpec(path, true)
		_, err := registryAccess.octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot access %v", path)
		}
	}

	// set credentials from pull secrets
	for _, secret := range secrets {
		if secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			continue
		}
		spec := dockerconfig.NewRepositorySpecForConfig(dockerConfigBytes)
		_, err := registryAccess.octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create credentials from secret")
		}
	}

	return registryAccess, nil
}

func (f *Factory) NewRegistryAccessFromOciOptions(ctx context.Context, log logr.Logger, fs vfs.FileSystem, allowPlainHttp bool, skipTLSVerify bool, registryConfigPath string, concourseConfigPath string, predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {
	octx := ocm.DefaultContext()

	// set available default credentials from dockerconfig files
	var spec *dockerconfig.RepositorySpec
	spec = dockerconfig.NewRepositorySpec(registryConfigPath, true)
	_, err := octx.CredentialsContext().RepositoryForSpec(spec)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot access %v", registryConfigPath)
	}

	return &RegistryAccess{
		octx: octx,
	}, nil
}

func (f *Factory) NewRegistryAccessForHelm(ctx context.Context, lsClient client.Client, contextObj *lsv1alpha1.Context, secrets []corev1.Secret, ociRegistryConfig *config.OCIConfiguration, sharedCache cache.Cache, ref *helmv1alpha1.RemoteChartReference) (model.RegistryAccess, error) {
	octx := ocm.DefaultContext()

	ociConfigFiles := make([]string, 0)
	if ociRegistryConfig != nil {
		ociConfigFiles = ociRegistryConfig.ConfigFiles
	}

	// set available default credentials from dockerconfig files
	var spec *dockerconfig.RepositorySpec
	for _, path := range ociConfigFiles {
		spec = dockerconfig.NewRepositorySpec(path, true)
		_, err := octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot access %v", path)
		}
	}

	// set credentials from pull secrets
	for _, secret := range secrets {
		if secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			continue
		}
		spec := dockerconfig.NewRepositorySpecForConfig(dockerConfigBytes)
		_, err := octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create credentials from secret")
		}
	}

	return &RegistryAccess{
		octx: octx,
	}, nil
}

func (f *Factory) NewOCIRegistryAccess(ctx context.Context, config *config.OCIConfiguration, cache cache.Cache, predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {
	//TODO implement me
	panic("implement me")
}

func (f *Factory) NewOCIRegistryAccessFromDockerAuthConfig(ctx context.Context, fs vfs.FileSystem, registrySecretBasePath string, predefinedComponentDescriptors ...*types.ComponentDescriptor) (model.RegistryAccess, error) {
	//TODO implement me
	panic("implement me")
}

func (f *Factory) NewOCITestRegistryAccess(address, username, password string) (model.RegistryAccess, error) {
	//TODO implement me
	panic("implement me")
}

func (f *Factory) NewLocalRegistryAccess(rootPath string) (model.RegistryAccess, error) {
	//TODO implement me
	panic("implement me")
}

func (f *Factory) NewHelmRepoResource(ctx context.Context, helmChartRepo *helmv1alpha1.HelmChartRepo, lsClient client.Client, contextObj *lsv1alpha1.Context) (model.Resource, error) {
	//TODO implement me
	panic("implement me")
}

func (f *Factory) NewHelmOCIResource(ctx context.Context, ociImageRef string, registryPullSecrets []corev1.Secret, ociConfig *config.OCIConfiguration, sharedCache cache.Cache) (model.Resource, error) {
	//TODO implement me
	panic("implement me")
}
