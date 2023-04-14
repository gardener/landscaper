package chartresolver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/osfs"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/deployer/helm/helmchartrepo"
	"github.com/gardener/landscaper/pkg/utils"
)

func newCnudieRegistryAccess(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache,
	ref *helmv1alpha1.RemoteChartReference) (model.RegistryAccess, error) {

	helmChartOCIResolver, err := newBlobResolverForChartsInOCIRegistry(ctx, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, err
	}

	helmChartRepoResolver, err := newBlobResolverForChartsInHelmChartRepo(lsClient, contextObj)
	if err != nil {
		return nil, err
	}

	registryAccess, err := cnudie.NewCnudieRegistry(ctx, registryPullSecrets, sharedCache, nil, ociConfig, ref.Inline,
		helmChartOCIResolver, helmChartRepoResolver)
	if err != nil {
		return nil, fmt.Errorf("unable to build registry access for helm charts: %w", err)
	}

	return registryAccess, nil
}

func newCnudieResourceInOCIRegistry(ctx context.Context, ociImageRef string, registryPullSecrets []corev1.Secret, ociConfig *config.OCIConfiguration, sharedCache cache.Cache) (model.Resource, error) {
	helmChartOCIResolver, err := newBlobResolverForChartsInOCIRegistry(ctx, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, err
	}

	ociAccess := cdv2.NewOCIRegistryAccess(ociImageRef)
	access, err := cdv2.NewUnstructured(ociAccess)
	if err != nil {
		return nil, fmt.Errorf("unable to construct ociClient registry access for %q: %w", ociImageRef, err)
	}

	res := cdv2.Resource{
		// only the type is needed other attributes can be ommitted.
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Type: HelmChartResourceType,
		},
		Relation: cdv2.ExternalRelation,
		Access:   &access,
	}

	return cnudie.NewResource(&res, helmChartOCIResolver), nil
}

func newCnudieResourceInHelmChartRepo(_ context.Context, helmChartRepo *helmv1alpha1.HelmChartRepo,
	lsClient client.Client, contextObj *lsv1alpha1.Context) (*cnudie.Resource, error) {

	access := helmchartrepo.HelmChartRepoAccess{
		ObjectType: cdv2.ObjectType{
			Type: helmchartrepo.HelmChartRepoType,
		},
		HelmChartRepo: *helmChartRepo,
	}

	raw, err := json.Marshal(access)
	if err != nil {
		return nil, fmt.Errorf("could not marshal helm chart repo data")
	}

	res := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Type: HelmChartResourceType,
		},
		Relation: cdv2.ExternalRelation,
		Access: &cdv2.UnstructuredTypedObject{
			ObjectType: cdv2.ObjectType{
				Type: helmchartrepo.HelmChartRepoType,
			},
			Raw: raw,
		},
	}

	helmChartRepoResolver, err := newBlobResolverForChartsInHelmChartRepo(lsClient, contextObj)
	if err != nil {
		return nil, err
	}

	return cnudie.NewResource(&res, helmChartRepoResolver), nil
}

func newBlobResolverForChartsInOCIRegistry(ctx context.Context, registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration, sharedCache cache.Cache) (ctf.TypedBlobResolver, error) {

	ociClient, err := createOCIClient(ctx,
		registryPullSecrets,
		ociConfig,
		sharedCache)
	if err != nil {
		return nil, fmt.Errorf("unable to build blob resolver for charts from oci registry: %w", err)
	}

	return NewHelmResolver(ociClient), nil
}

func createOCIClient(ctx context.Context, registryPullSecrets []corev1.Secret, ociConfig *config.OCIConfiguration, sharedCache cache.Cache) (ociclient.Client, error) {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "helmDeployerController.createOCIClient"})

	// always add an oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if ociConfig != nil {
		ociConfigFiles = ociConfig.ConfigFiles
	}
	ociKeyring, err := credentials.NewBuilder(logger.WithName("ociKeyring").Logr()).
		WithFS(osfs.New()).
		FromConfigFiles(ociConfigFiles...).
		FromPullSecrets(registryPullSecrets...).
		Build()
	if err != nil {
		return nil, err
	}
	ociClient, err := ociclient.NewClient(logger.Logr(),
		utils.WithConfiguration(ociConfig),
		ociclient.WithKeyring(ociKeyring),
		ociclient.WithCache(sharedCache),
	)
	if err != nil {
		return nil, err
	}

	return ociClient, nil
}

func newBlobResolverForChartsInHelmChartRepo(lsClient client.Client, contextObj *lsv1alpha1.Context) (ctf.TypedBlobResolver, error) {
	helmChartRepoClient, err := helmchartrepo.NewHelmChartRepoClient(contextObj, lsClient)
	if err != nil {
		return nil, fmt.Errorf("unable to build blob resolver for charts from helm chart repos: %w", err)
	}

	return helmchartrepo.NewHelmChartRepoResolver(helmChartRepoClient), nil
}
