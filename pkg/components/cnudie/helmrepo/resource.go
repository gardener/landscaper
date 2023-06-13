package helmrepo

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

const (
	// HelmChartResourceType describes the helm resource type of a component descriptor defined resource.
	HelmChartResourceType = types.HelmChartResourceType

	// HelmChartRepoType is the access type of a helm chart repository.
	HelmChartRepoType = "helmChartRepository"
)

func NewResourceDataForHelmRepo(helmChartRepo *helmv1alpha1.HelmChartRepo) (*types.Resource, error) {
	access := HelmChartRepoAccess{
		ObjectType: cdv2.ObjectType{
			Type: HelmChartRepoType,
		},
		HelmChartRepo: *helmChartRepo,
	}

	raw, err := json.Marshal(access)
	if err != nil {
		return nil, fmt.Errorf("could not marshal helm chart repo data")
	}

	return &types.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Type: HelmChartResourceType,
		},
		Relation: cdv2.ExternalRelation,
		Access: &types.UnstructuredTypedObject{
			ObjectType: cdv2.ObjectType{
				Type: HelmChartRepoType,
			},
			Raw: raw,
		},
	}, nil
}

type HelmChartRepoAccess struct {
	cdv2.ObjectType            `json:",inline"`
	helmv1alpha1.HelmChartRepo `json:",inline"`
}

func (a *HelmChartRepoAccess) GetType() string {
	return HelmChartRepoType
}
