package helmchartrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/pkg/deployer/helm/shared"
)

type HelmChartRepoResolver struct {
	helmChartRepoClient *HelmChartRepoClient
}

func NewHelmChartRepoResolver(helmChartRepoClient *HelmChartRepoClient) ctf.TypedBlobResolver {
	return NewHelmChartRepoResolverAsHelmChartRepoResolver(helmChartRepoClient)
}

func NewHelmChartRepoResolverAsHelmChartRepoResolver(helmChartRepoClient *HelmChartRepoClient) *HelmChartRepoResolver {
	return &HelmChartRepoResolver{
		helmChartRepoClient: helmChartRepoClient,
	}
}

func (h *HelmChartRepoResolver) CanResolve(res cdv2.Resource) bool {
	if res.GetType() != shared.HelmChartResourceType && res.GetType() != shared.OldHelmResourceType {
		return false
	}
	return res.Access != nil && res.Access.GetType() == HelmChartRepoType
}

func (h *HelmChartRepoResolver) Info(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error) {
	return h.Resolve(ctx, res, nil)
}

func (h *HelmChartRepoResolver) ResolveHelmChart(_ context.Context, _ io.Writer) (*ctf.BlobInfo, error) {
	return nil, errors.New("no helm chart repo data provided")
}

func (h *HelmChartRepoResolver) Resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	helmChartRepoAccess := &HelmChartRepoAccess{}

	if err := json.Unmarshal(res.Access.Raw, helmChartRepoAccess); err != nil {
		return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	return h.ResolveHelmChart(ctx, writer)
}
