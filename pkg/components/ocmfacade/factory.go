package ocmfacade

import (
	"context"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

	octx := ocm.DefaultContext()

	ociConfigFiles := make([]string, 0)
	if ociRegistryConfig != nil {
		ociConfigFiles = ociRegistryConfig.ConfigFiles
	}

	var spec *dockerconfig.RepositorySpec
	for _, path := range ociConfigFiles {
		spec = dockerconfig.NewRepositorySpec(path, true)
		_, err := octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot access %v", path)
		}
	}

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
