// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"fmt"

	credconfig "github.com/open-component-model/ocm/pkg/contexts/credentials/config"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ocmcommon "github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/helm/identity"
	"github.com/open-component-model/ocm/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/components/common"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmlib/inlinecompdesc"
	"github.com/gardener/landscaper/pkg/components/ocmlib/repository"
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

	if fs == nil {
		fs = osfs.New()
	}

	registryAccess := &RegistryAccess{}
	registryAccess.octx = ocm.New(datacontext.MODE_EXTENDED)
	registryAccess.session = ocm.NewSession(datacontext.NewSession())

	var localfs vfs.FileSystem
	if localRegistryConfig != nil {
		var err error
		localfs, err = projectionfs.New(fs, localRegistryConfig.RootPath)
		if err != nil {
			return nil, err
		}
		vfsattr.Set(registryAccess.octx, localfs)
	} else {
		// safe guard that the file system cannot be accessed
		vfsattr.Set(registryAccess.octx, readonlyfs.New(memoryfs.New()))
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

		provider := repository.NewMemoryCompDescProvider(descriptors)
		registryAccess.inlineRepository, err = repository.NewRepository(registryAccess.octx, provider, localfs)
		if err != nil {
			return nil, err
		}
		_ = registryAccess.session.AddCloser(registryAccess.inlineRepository)
		if err != nil {
			return nil, err
		}
		registryAccess.resolver = registryAccess.inlineRepository
		if len(inlineCd.RepositoryContexts) > 0 {
			repoCtx := inlineCd.GetEffectiveRepositoryContext()
			registryAccess.inlineSpec, err = registryAccess.octx.RepositorySpecForConfig(repoCtx.Raw, nil)
			if err != nil {
				return nil, err
			}
			registryAccess.resolver, err = registryAccess.session.LookupRepository(registryAccess.octx, registryAccess.inlineSpec)
			if err != nil {
				return nil, err
			}
			registryAccess.resolver = ocm.NewCompoundResolver(registryAccess.inlineRepository, registryAccess.resolver)
		}

	}

	if ociRegistryConfig != nil {
		// set credentials from pull secrets
		if err := addConfigFileCredsToCredContext(fs, ociRegistryConfig.ConfigFiles, registryAccess.octx); err != nil {
			return nil, err
		}
	}

	// set credentials from pull secrets
	if err := addSecretCredsToCredContext(secrets, registryAccess.octx); err != nil {
		return nil, err
	}

	return registryAccess, nil
}

func (f *Factory) NewHelmRepoResource(ctx context.Context, helmChartRepo *helmv1alpha1.HelmChartRepo, lsClient client.Client, contextObj *lsv1alpha1.Context) (model.TypedResourceProvider, error) {
	provider := &HelmChartProvider{
		ocictx:  oci.New(datacontext.MODE_EXTENDED),
		ref:     helmChartRepo.HelmChartName,
		version: helmChartRepo.HelmChartVersion,
		repourl: common.NormalizeUrl(helmChartRepo.HelmChartRepoUrl),
	}

	if contextObj != nil && contextObj.Configurations != nil {
		if rawAuths, ok := contextObj.Configurations[helmv1alpha1.HelmChartRepoCredentialsKey]; ok {
			repoCredentials := helmv1alpha1.HelmChartRepoCredentials{}
			err := yaml.Unmarshal(rawAuths.RawMessage, &repoCredentials)
			if err != nil {
				return nil, lserrors.NewWrappedError(err, "NewHelmChartRepoClient", "ParsingAuths", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
			}

			for _, a := range repoCredentials.Auths {
				id := identity.GetConsumerId(provider.repourl, "")
				provider.ocictx.CredentialsContext().SetCredentialsForConsumer(id, &CredentialSource{
					lsClient:  lsClient,
					auth:      a,
					namespace: contextObj.Namespace,
				})
			}
		}
	}

	return provider, nil
}

type CredentialSource struct {
	lsClient  client.Client
	auth      helmv1alpha1.Auth
	namespace string
}

func (c *CredentialSource) Credentials(ctx credentials.Context, _ ...credentials.CredentialsSource) (credentials.Credentials, error) {
	authheader, err := common.GetAuthHeader(context.Background(), &c.auth, c.lsClient, c.namespace)
	if err != nil {
		return nil, err
	}
	username, password, ok := common.ParseBasicAuth(authheader)
	if !ok {
		return nil, errors.New("only basic auth supported")
	}

	props := ocmcommon.Properties{
		identity.ATTR_USERNAME: username,
		identity.ATTR_PASSWORD: password,
	}
	if c.auth.CustomCAData != "" {
		props[identity.ATTR_CERTIFICATE_AUTHORITY] = c.auth.CustomCAData
	}
	return credentials.NewCredentials(props), nil
}

func (f *Factory) NewHelmOCIResource(ctx context.Context,
	fs vfs.FileSystem, ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (model.TypedResourceProvider, error) {

	if fs == nil {
		fs = osfs.New()
	}

	refspec, err := oci.ParseRef(ociImageRef)
	if err != nil {
		return nil, err
	}

	provider := &HelmChartProvider{
		ocictx:  oci.New(datacontext.MODE_EXTENDED),
		ref:     refspec.Repository,
		version: refspec.Version(),
		repourl: fmt.Sprintf("oci://%s", refspec.Host),
	}

	if ociConfig != nil {
		// set credentials from config files
		if err = addConfigFileCredsToCredContext(fs, ociConfig.ConfigFiles, provider.ocictx); err != nil {
			return nil, err
		}
	}

	// set credentials from pull secrets
	if err = addSecretCredsToCredContext(registryPullSecrets, provider.ocictx); err != nil {
		return nil, err
	}

	return provider, nil
}

func addConfigFileCredsToCredContext(fs vfs.FileSystem, filePaths []string, provider credentials.ContextProvider) error {
	credctx := provider.CredentialsContext()

	// set available default credentials from dockerconfig files
	for _, path := range filePaths {
		dockerConfigBytes, err := vfs.ReadFile(fs, path)
		if err != nil {
			return err
		}
		spec := dockerconfig.NewRepositorySpecForConfig(dockerConfigBytes, true)
		_, err = credctx.RepositoryForSpec(spec)
		if err != nil {
			return err
		}
	}
	return nil
}

func addSecretCredsToCredContext(secrets []corev1.Secret, provider credentials.ContextProvider) error {
	credctx := provider.CredentialsContext()
	cfgctx := credctx.ConfigContext()

	for _, secret := range secrets {
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if ok {
			spec := dockerconfig.NewRepositorySpecForConfig(dockerConfigBytes, true)
			_, err := credctx.RepositoryForSpec(spec)
			if err != nil {
				return errors.Wrapf(err, "cannot create credentials from secret")
			}
		}
		ocmConfigBytes, ok := secret.Data[".ocmcredentialconfig"]
		if ok {
			cfg, err := cfgctx.GetConfigForData(ocmConfigBytes, runtime.DefaultYAMLEncoding)
			if err != nil {
				return err
			}
			if cfg.GetKind() == credconfig.ConfigType {
				err := cfgctx.ApplyConfig(cfg, fmt.Sprintf("landscaper secret: %s/%s", secret.Namespace, secret.Name))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
