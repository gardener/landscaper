package helmchartrepo

import (
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

// HelmChartRepoType is the access type of a helm chart repository.
const HelmChartRepoType = "helmChartRepository"

type HelmChartRepoAccess struct {
	v2.ObjectType          `json:",inline"`
	v1alpha1.HelmChartRepo `json:",inline"`
}

func (a *HelmChartRepoAccess) GetType() string {
	return HelmChartRepoType
}
