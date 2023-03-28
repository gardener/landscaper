package helmoci

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/deployer/helm/shared"
)

const (
	// HelmChartConfigMediaType is the reserved media type for the Helm chart manifest config
	HelmChartConfigMediaType = "application/vnd.cncf.helm.config.v1+json"

	// ConfigMediaType is the reserved media type for the Helm chart manifest config
	ConfigMediaType = "application/vnd.cncf.helm.config.v1+json"

	// ChartLayerMediaType is the reserved media type for Helm chart package content
	ChartLayerMediaType = "application/vnd.cncf.helm.chart.content.v1.tar+gzip"

	// ProvLayerMediaType is the reserved media type for Helm chart provenance files
	ProvLayerMediaType = "application/vnd.cncf.helm.chart.provenance.v1.prov"

	// LegacyChartLayerMediaType is the legacy reserved media type for Helm chart package content.
	LegacyChartLayerMediaType = "application/tar+gzip"

	// OldHelmResourceType describes the old helm resource type of a component descrptor defined resource.
	OldHelmResourceType = shared.OldHelmResourceType
	// HelmChartResourceType describes the helm resource type of a component descrptor defined resource.
	HelmChartResourceType = shared.HelmChartResourceType
)

func NewResourceDataForHelmOCI(ociImageRef string) (*types.Resource, error) {
	ociAccess := cdv2.NewOCIRegistryAccess(ociImageRef)
	access, err := cdv2.NewUnstructured(ociAccess)
	if err != nil {
		return nil, fmt.Errorf("unable to construct helm oci access data for %s: %w", ociImageRef, err)
	}

	return &types.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Type: HelmChartResourceType,
		},
		Relation: cdv2.ExternalRelation,
		Access:   &access,
	}, nil
}
