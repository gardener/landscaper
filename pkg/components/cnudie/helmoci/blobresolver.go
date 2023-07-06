package helmoci

import (
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/osfs"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	cnudieutils "github.com/gardener/landscaper/pkg/components/cnudie/utils"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type BlobResolverForHelmOCI struct {
	ociClient ociclient.Client
}

var _ ctf.TypedBlobResolver = &BlobResolverForHelmOCI{}

// NewBlobResolverForHelmOCI returns a BlobResolver for helm charts that are stored in an OCI registry.
func NewBlobResolverForHelmOCI(ctx context.Context,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (ctf.TypedBlobResolver, error) {

	ociClient, err := createOCIClient(ctx, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, fmt.Errorf("unable to build blob resolver for charts from oci registry: %w", err)
	}

	return &BlobResolverForHelmOCI{
		ociClient: ociClient,
	}, nil
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
		cnudieutils.WithConfiguration(ociConfig),
		ociclient.WithKeyring(ociKeyring),
		ociclient.WithCache(sharedCache),
	)
	if err != nil {
		return nil, err
	}

	return ociClient, nil
}

func (h BlobResolverForHelmOCI) CanResolve(res types.Resource) bool {
	if res.GetType() != types.HelmChartResourceType && res.GetType() != types.OldHelmResourceType {
		return false
	}
	return res.Access != nil && res.Access.GetType() == cdv2.OCIRegistryType
}

func (h BlobResolverForHelmOCI) Info(ctx context.Context, res types.Resource) (*types.BlobInfo, error) {
	return h.resolve(ctx, res, nil)
}

func (h BlobResolverForHelmOCI) Resolve(ctx context.Context, res types.Resource, writer io.Writer) (*types.BlobInfo, error) {
	return h.resolve(ctx, res, writer)
}

func (h BlobResolverForHelmOCI) resolve(ctx context.Context, res types.Resource, writer io.Writer) (*types.BlobInfo, error) {
	ociArtifactAccess := &cdv2.OCIRegistryAccess{}
	if err := cdv2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, ociArtifactAccess); err != nil {
		return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	manifest, err := h.ociClient.GetManifest(ctx, ociArtifactAccess.ImageReference)
	if err != nil {
		return nil, err
	}

	// since helm 3.7 two layers are valid.
	// one containing the chart and one containing the provenance files.
	if len(manifest.Layers) > 2 || len(manifest.Layers) == 0 {
		return nil, fmt.Errorf("expected one or two layers but found %d", len(manifest.Layers))
	}

	if chartLayers := ociclient.GetLayerByMediaType(manifest.Layers, ChartLayerMediaType); len(chartLayers) != 0 {
		if manifest.Config.MediaType != HelmChartConfigMediaType {
			return nil, fmt.Errorf("unexpected media type of helm config. Expected %s but got %s", HelmChartConfigMediaType, manifest.Config.MediaType)
		}
		chartLayer := chartLayers[0]
		// todo: check verify the signature if a provenance layer is present.
		if writer != nil {
			if err := h.ociClient.Fetch(ctx, ociArtifactAccess.ImageReference, chartLayer, writer); err != nil {
				return nil, err
			}
		}
		return &types.BlobInfo{
			MediaType: ChartLayerMediaType,
			Digest:    chartLayer.Digest.String(),
			Size:      chartLayer.Size,
		}, nil
	}
	if legacyChartLayers := ociclient.GetLayerByMediaType(manifest.Layers, LegacyChartLayerMediaType); len(legacyChartLayers) != 0 {
		logging.FromContextOrDiscard(ctx).Info("LEGACY Helm Chart used", "ref", ociArtifactAccess.ImageReference)
		chartLayer := legacyChartLayers[0]
		if writer != nil {
			if err := h.ociClient.Fetch(ctx, ociArtifactAccess.ImageReference, chartLayer, writer); err != nil {
				return nil, err
			}
		}
		return &types.BlobInfo{
			MediaType: LegacyChartLayerMediaType,
			Digest:    chartLayer.Digest.String(),
			Size:      chartLayer.Size,
		}, nil
	}

	return nil, fmt.Errorf("unknown oci artifact of type %s", manifest.Config.MediaType)
}
