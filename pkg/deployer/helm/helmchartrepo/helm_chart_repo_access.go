package helmchartrepo

import (
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// HelmChartRepoType is the access type of a helm chart repository.
const HelmChartRepoType = "helmChartRepository"

type HelmChartRepoAccess struct {
	v2.ObjectType `json:",inline"`
}

func (a *HelmChartRepoAccess) GetType() string {
	return HelmChartRepoType
}
