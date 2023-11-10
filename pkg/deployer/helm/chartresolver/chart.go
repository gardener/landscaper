// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/gardener/component-cli/ociclient/cache"
	"helm.sh/helm/v3/pkg/chart"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/registries"
)

// NoChartDefinedError is the error that is returned if no Helm chart was provided
var NoChartDefinedError = errors.New("no chart was provided")

// GetChart resolves the chart based on a chart access configuration.
func GetChart(ctx context.Context,
	chartConfig *helmv1alpha1.Chart,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (*chart.Chart, error) {

	if chartConfig.Archive != nil {
		return getChartFromArchive(chartConfig.Archive)
	}

	if len(chartConfig.Ref) != 0 {
		return getChartFromOCIRef(ctx, contextObj, chartConfig.Ref, registryPullSecrets, ociConfig, sharedCache)
	}

	if chartConfig.HelmChartRepo != nil {
		return getChartFromHelmChartRepo(ctx, lsClient, contextObj, chartConfig.HelmChartRepo)
	}

	if chartConfig.FromResource != nil {
		return nil, errors.New("chart.fromResource is no longer supported")
	}
	return nil, NoChartDefinedError
}

func getChartFromArchive(archiveConfig *helmv1alpha1.ArchiveAccess) (*chart.Chart, error) {
	if len(archiveConfig.Raw) != 0 {
		data, err := base64.StdEncoding.DecodeString(archiveConfig.Raw)
		if err != nil {
			return nil, fmt.Errorf("unable to decode helm archive: %w", err)
		}
		ch, err := chartloader.LoadArchive(bytes.NewBuffer(data))
		if err != nil {
			return nil, fmt.Errorf("unable to load chart from archive: %w", err)
		}
		return ch, err
	}
	if archiveConfig.Remote != nil {
		res, err := http.Get(archiveConfig.Remote.URL)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch helm chart from %q: %w", archiveConfig.Remote.URL, err)
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, fmt.Errorf("unable to fetch helm chart from %q: %s", archiveConfig.Remote.URL, res.Status)
		}
		ch, err := chartloader.LoadArchive(res.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to load chart from %q: %w", archiveConfig.Remote.URL, err)
		}
		if err := res.Body.Close(); err != nil {
			return nil, fmt.Errorf("unable to close remote stream from %q: %w", archiveConfig.Remote.URL, err)
		}
		return ch, err
	}
	return nil, NoChartDefinedError
}

func getChartFromOCIRef(ctx context.Context,
	contextObj *lsv1alpha1.Context,
	ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (*chart.Chart, error) {

	resource, err := registries.GetFactory(contextObj.UseOCM).NewHelmOCIResource(ctx, nil, ociImageRef, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, err
	}

	resourceContent, err := resource.GetTypedContent(ctx)
	if err != nil {
		return nil, err
	}
	content, ok := resourceContent.Resource.(*chart.Chart)
	if !ok {
		return nil, fmt.Errorf("received resource of type %T but expected type *Chart", content)
	}
	return content, nil
}

func getChartFromHelmChartRepo(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	repo *helmv1alpha1.HelmChartRepo) (*chart.Chart, error) {

	resource, err := registries.GetFactory(contextObj.UseOCM).NewHelmRepoResource(ctx, repo, lsClient, contextObj)
	if err != nil {
		return nil, fmt.Errorf("unable to construct resource for chart %q with version %q from helm chart repo %q: %w",
			repo.HelmChartName, repo.HelmChartVersion, repo.HelmChartRepoUrl, err)
	}

	resourceContent, err := resource.GetTypedContent(ctx)
	if err != nil {
		return nil, err
	}
	content, ok := resourceContent.Resource.(*chart.Chart)
	if !ok {
		return nil, fmt.Errorf("received resource of type %T but expected type *Chart", content)
	}
	return content, err
}
