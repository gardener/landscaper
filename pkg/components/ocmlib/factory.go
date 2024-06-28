// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"fmt"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ocmcommon "github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	helmid "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity"
	credconfig "github.com/open-component-model/ocm/pkg/contexts/credentials/config"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	ocmutils "github.com/open-component-model/ocm/pkg/contexts/ocm/utils"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/common"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmlib/inlinecompdesc"
	"github.com/gardener/landscaper/pkg/components/ocmlib/repository"
	"github.com/gardener/landscaper/pkg/utils"
)

type Factory struct{}

var _ model.Factory = &Factory{}

func (*Factory) NewRegistryAccess(ctx context.Context, options *model.RegistryAccessOptions) (model.RegistryAccess, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "CreateRegistryAccess")
	defer pm.StopDebug()

	fs := options.Fs
	if fs == nil {
		fs = osfs.New()
	}

	registryAccess := &RegistryAccess{}
	registryAccess.octx = ocm.FromContext(ctx)
	registryAccess.octx.Finalizer().Close(registryAccess)
	registryAccess.session = ocm.NewSession(datacontext.NewSession())

	// If a config map containing the data of an ocm config file is provided, apply its configuration.
	if err := ApplyOCMConfigMapToOCMContext(registryAccess.octx, options.OcmConfig); err != nil {
		return nil, err
	}

	if err := applyAdditionalRepositoryContexts(registryAccess.octx, options.AdditionalRepositoryContexts); err != nil {
		return nil, err
	}

	registryAccess.Overwriter = options.Overwriter

	// If a local registry configuration is provided, the vfsattr (= virtual file system attribute) in the ocm context's
	// data context is set to projection of the hosts file system which has the specified path as its root.
	// This attribute is used when creating a type "local" ocm repository (a special repository implementation used by the
	// landscaper) from spec.
	// For more details, check pkg/components/ocmlib/repository and pkg/components/ocmlib/repository/local.
	var localfs vfs.FileSystem
	if options.LocalRegistryConfig != nil {
		var err error
		localfs, err = projectionfs.New(fs, options.LocalRegistryConfig.RootPath)
		if err != nil {
			return nil, err
		}
		vfsattr.Set(registryAccess.octx, localfs)
	} else {
		// safe guard that the file system cannot be accessed
		vfsattr.Set(registryAccess.octx, readonlyfs.New(memoryfs.New()))
	}

	// If an inline component descriptor is provided in the installation, the inline component descriptor is resolved
	// to a flat list of component descriptors (see pkg/components/ocmlib/inlinecompdesc/util.go expand documentation
	// for why this is necessary). This list is then used to create a type "inline" ocm repository (a special repository
	// implementation used by the landscaper).
	// For more details, check pkg/components/ocmlib/repository and pkg/components/ocmlib/repository/inline.
	// The references of inline component descriptors can reference either
	// 1) another component descriptor nested in the inline component descriptor or
	// 2) a component descriptor in the repository context of the top level inline component descriptor itself
	// Thus, a compound resolver consisting of the inline repository and the repository specified by the repository
	// context is added to the registry access. This resolver is used if the repository context in the
	// component descriptor reference is equal to the repository context of the inline component descriptor.
	if options.InlineCd != nil {
		cd, err := runtime.DefaultYAMLEncoding.Marshal(options.InlineCd)
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

		registryAccess.resolver = registryAccess.inlineRepository
		if len(options.InlineCd.RepositoryContexts) > 0 {
			repoCtx := options.InlineCd.GetEffectiveRepositoryContext()
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

	if options.OciRegistryConfig != nil {
		// set credentials from pull secrets
		if err := addConfigFileCredsToCredContext(fs, options.OciRegistryConfig.ConfigFiles, registryAccess.octx); err != nil {
			return nil, err
		}
	}

	// set credentials from pull secrets
	if err := AddSecretCredsToCredContext(options.Secrets, registryAccess.octx); err != nil {
		return nil, err
	}

	return registryAccess, nil
}

func (f *Factory) CreateRegistryAccess(ctx context.Context,
	fs vfs.FileSystem,
	ocmconfig *corev1.ConfigMap,
	secrets []corev1.Secret,
	localRegistryConfig *config.LocalRegistryConfiguration,
	ociRegistryConfig *config.OCIConfiguration,
	inlineCd *types.ComponentDescriptor,
	_ ...ctf.TypedBlobResolver) (model.RegistryAccess, error) {

	return f.NewRegistryAccess(ctx, &model.RegistryAccessOptions{
		Fs:                  fs,
		OcmConfig:           ocmconfig,
		Secrets:             secrets,
		LocalRegistryConfig: localRegistryConfig,
		OciRegistryConfig:   ociRegistryConfig,
		InlineCd:            inlineCd,
	})
}

func (f *Factory) NewHelmRepoResource(ctx context.Context, ocmconfig *corev1.ConfigMap, helmChartRepo *helmv1alpha1.HelmChartRepo, lsClient client.Client, contextObj *lsv1alpha1.Context) (model.TypedResourceProvider, error) {
	octx := ocm.FromContext(ctx)
	if err := ApplyOCMConfigMapToOCMContext(octx, ocmconfig); err != nil {
		return nil, err
	}

	provider := &HelmChartProvider{
		ocictx:  octx.OCIContext(),
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
				id := helmid.GetConsumerId(provider.repourl, "")
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
func NewHelmCredentialSource(lsClient client.Client, auth helmv1alpha1.Auth, namespace string) *CredentialSource {
	return &CredentialSource{
		lsClient:  lsClient,
		auth:      auth,
		namespace: namespace,
	}
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
		helmid.ATTR_USERNAME: username,
		helmid.ATTR_PASSWORD: password,
	}
	if c.auth.CustomCAData != "" {
		props[helmid.ATTR_CERTIFICATE_AUTHORITY] = c.auth.CustomCAData
	}
	return credentials.NewCredentials(props), nil
}

func (f *Factory) NewHelmOCIResource(ctx context.Context,
	fs vfs.FileSystem,
	ocmconfig *corev1.ConfigMap,
	ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration) (model.TypedResourceProvider, error) {

	octx := ocm.FromContext(ctx)
	if err := ApplyOCMConfigMapToOCMContext(octx, ocmconfig); err != nil {
		return nil, err
	}

	if fs == nil {
		fs = osfs.New()
	}

	refspec, err := oci.ParseRef(ociImageRef)
	if err != nil {
		return nil, err
	}

	provider := &HelmChartProvider{
		ocictx:  octx.OCIContext(),
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
	if err = AddSecretCredsToCredContext(registryPullSecrets, provider.ocictx); err != nil {
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

func AddSecretCredsToCredContext(secrets []corev1.Secret, provider credentials.ContextProvider) error {
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

func ApplyOCMConfigMapToOCMContext(octx ocm.Context, ocmconfig *corev1.ConfigMap) error {
	if ocmconfig != nil {
		ocmconfigdata, ok := ocmconfig.Data[ocmutils.DEFAULT_OCM_CONFIG]
		if !ok {
			return fmt.Errorf("ocm configuration config map does not contain key \"%s\"", ocmutils.DEFAULT_OCM_CONFIG)
		}
		if len(ocmconfigdata) > 0 {
			cfg, err := octx.ConfigContext().GetConfigForData([]byte(ocmconfigdata), nil)
			if err != nil {
				return fmt.Errorf("invalid ocm config in \"%s\" in namespace \"%s\": %w", ocmconfig.Name, ocmconfig.Namespace, err)
			}
			err = octx.ConfigContext().ApplyConfig(cfg, fmt.Sprintf("%s/%s", ocmconfig.Namespace, ocmconfig.Name))
			if err != nil {
				return fmt.Errorf("cannot apply ocm config in \"%s\" in namespace \"%s\": %w", ocmconfig.Name, ocmconfig.Namespace, err)
			}
		}
	}
	return nil
}

func applyAdditionalRepositoryContexts(octx ocm.Context, additionalRepositoryContexts []types.PrioritizedRepositoryContext) error {
	for i := range additionalRepositoryContexts {
		a := &additionalRepositoryContexts[i]
		spec, err := octx.RepositorySpecForConfig(a.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
		if err != nil {
			return err
		}
		octx.AddResolverRule("", spec, a.Priority)
	}
	return nil
}
