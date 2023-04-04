package helmchartrepo

import (
	"context"
	"errors"
	"io"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type HelmChartRepoResource struct {
	helmChartRepo         *v1alpha1.HelmChartRepo
	helmChartRepoResolver *HelmChartRepoResolver
}

var _ model.Resource = &HelmChartRepoResource{}

func NewHelmChartRepoResource(_ context.Context,
	helmChartRepo *v1alpha1.HelmChartRepo,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context) (*HelmChartRepoResource, error) {

	helmChartRepoClient, lsError := NewHelmChartRepoClient(contextObj, lsClient)
	if lsError != nil {
		return nil, lsError
	}

	helmChartRepoResolver := NewHelmChartRepoResolverAsHelmChartRepoResolver(helmChartRepoClient)

	return &HelmChartRepoResource{
		helmChartRepo:         helmChartRepo,
		helmChartRepoResolver: helmChartRepoResolver,
	}, nil
}

func (h HelmChartRepoResource) GetName() string {
	return ""
}

func (h HelmChartRepoResource) GetVersion() string {
	return ""
}

func (h HelmChartRepoResource) GetDescriptor(ctx context.Context) ([]byte, error) {
	return nil, errors.New("method GetDescriptor is not supported by HelmChartRepoResource")
}

func (h HelmChartRepoResource) GetBlob(ctx context.Context, writer io.Writer) error {
	_, err := h.helmChartRepoResolver.ResolveHelmChart(ctx, h.helmChartRepo, writer)
	return err
}

func (h HelmChartRepoResource) GetBlobInfo(ctx context.Context) (*model.BlobInfo, error) {
	return nil, errors.New("method GetBlobInfo is not supported by HelmChartRepoResource")
}
