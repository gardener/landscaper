package helmchartrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"helm.sh/helm/v3/pkg/repo"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
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

func (h *HelmChartRepoResolver) ResolveHelmChart(ctx context.Context, helmChartRepo *v1alpha1.HelmChartRepo, writer io.Writer) (*ctf.BlobInfo, error) {
	if helmChartRepo.HelmChartRepoUrl == "" {
		return nil, errors.New("no helm chart repo url provided")
	}

	helmChartRepoUrl := normalizeUrl(helmChartRepo.HelmChartRepoUrl) + "/index.yaml"

	repoCatalog, err := h.helmChartRepoClient.fetchRepoCatalog(ctx, helmChartRepoUrl)
	if err != nil {
		return nil, err
	}

	chartURL, err := h.findChartInRepoCatalog(repoCatalog, helmChartRepoUrl, helmChartRepo.HelmChartName, helmChartRepo.HelmChartVersion)
	if err != nil {
		return nil, err
	}

	chartBytes, err := h.helmChartRepoClient.fetchChart(ctx, chartURL)
	if err != nil {
		return nil, err
	}

	if writer != nil {
		_, err := writer.Write(chartBytes)
		if err != nil {
			return nil, err
		}
	}

	return &ctf.BlobInfo{
		MediaType: "",
		Digest:    "",
		Size:      int64(len(chartBytes)),
	}, nil
}

func (h *HelmChartRepoResolver) Resolve(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	helmChartRepoAccess := &HelmChartRepoAccess{}

	if err := json.Unmarshal(res.Access.Raw, helmChartRepoAccess); err != nil {
		return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	return h.ResolveHelmChart(ctx, &helmChartRepoAccess.HelmChartRepo, writer)
}

// findChartInRepoIndex returns the URL of a chart given a Helm repository and its name and version
func (h *HelmChartRepoResolver) findChartInRepoCatalog(repoCatalog *repo.IndexFile, repoURL, chartName, chartVersion string) (string, error) {
	errMsg := fmt.Sprintf("chart %q", chartName)
	if chartVersion != "" {
		errMsg = fmt.Sprintf("%s version %q", errMsg, chartVersion)
	}
	cv, err := repoCatalog.Get(chartName, chartVersion)
	if err != nil {
		return "", fmt.Errorf("%s not found in repository", errMsg)
	}
	if len(cv.URLs) == 0 {
		return "", fmt.Errorf("%s has no downloadable URLs", errMsg)
	}
	return h.resolveChartURL(repoURL, cv.URLs[0])
}

func (h *HelmChartRepoResolver) resolveChartURL(index, chartName string) (string, error) {
	indexURL, err := url.Parse(strings.TrimSpace(index))
	if err != nil {
		return "", fmt.Errorf("could not parse chart url: %w", err)
	}
	chartURL, err := indexURL.Parse(strings.TrimSpace(chartName))
	if err != nil {
		return "", fmt.Errorf("could not parse chart url: %w", err)
	}
	return chartURL.String(), nil
}
