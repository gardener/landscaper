// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helmrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gardener/landscaper/pkg/components/common"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type BlobResolverForHelmRepo struct {
	helmChartRepoClient *HelmChartRepoClient
}

var _ ctf.TypedBlobResolver = &BlobResolverForHelmRepo{}

// NewBlobResolverForHelmRepo returns a BlobResolver for helm charts that are stored in a helm chart repository.
func NewBlobResolverForHelmRepo(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context) (ctf.TypedBlobResolver, error) {

	helmChartRepoClient, err := NewHelmChartRepoClient(contextObj, lsClient)
	if err != nil {
		return nil, fmt.Errorf("unable to build blob resolver for charts from helm chart repos: %w", err)
	}

	return &BlobResolverForHelmRepo{
		helmChartRepoClient: helmChartRepoClient,
	}, nil
}

func (h *BlobResolverForHelmRepo) CanResolve(res types.Resource) bool {
	if res.GetType() != types.HelmChartResourceType && res.GetType() != types.OldHelmResourceType {
		return false
	}
	return res.Access != nil && res.Access.GetType() == HelmChartRepoType
}

func (h *BlobResolverForHelmRepo) Info(ctx context.Context, res types.Resource) (*types.BlobInfo, error) {
	return h.Resolve(ctx, res, nil)
}

func (h *BlobResolverForHelmRepo) ResolveHelmChart(ctx context.Context, helmChartRepo *v1alpha1.HelmChartRepo, writer io.Writer) (*types.BlobInfo, error) {
	if helmChartRepo.HelmChartRepoUrl == "" {
		return nil, errors.New("no helm chart repo url provided")
	}

	helmChartRepoUrl := common.NormalizeUrl(helmChartRepo.HelmChartRepoUrl) + "/index.yaml"

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

	return &types.BlobInfo{
		MediaType: "",
		Digest:    "",
		Size:      int64(len(chartBytes)),
	}, nil
}

func (h *BlobResolverForHelmRepo) Resolve(ctx context.Context, res types.Resource, writer io.Writer) (*types.BlobInfo, error) {
	helmChartRepoAccess := &HelmChartRepoAccess{}

	if err := json.Unmarshal(res.Access.Raw, helmChartRepoAccess); err != nil {
		return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	return h.ResolveHelmChart(ctx, &helmChartRepoAccess.HelmChartRepo, writer)
}

// findChartInRepoIndex returns the URL of a chart given a Helm repository and its name and version
func (h *BlobResolverForHelmRepo) findChartInRepoCatalog(repoCatalog *repo.IndexFile, repoURL, chartName, chartVersion string) (string, error) {
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

func (h *BlobResolverForHelmRepo) resolveChartURL(index, chartName string) (string, error) {
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
