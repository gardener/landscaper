package helmchartrepo

import (
	"github.com/go-logr/logr"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type HelmChartRepoClient struct {
	log logr.Logger
}

func NewHelmChartRepoClient(log logr.Logger, context *lsv1alpha1.Context) (*HelmChartRepoClient, error) {
	return &HelmChartRepoClient{
		log: log,
	}, nil
}
