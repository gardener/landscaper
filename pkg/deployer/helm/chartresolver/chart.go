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
	"github.com/gardener/landscaper/pkg/deployerlegacy"
	"net/http"

	"github.com/gardener/component-cli/ociclient/cache"
	"helm.sh/helm/v3/pkg/chart"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
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
		return getChartFromOCIRef(ctx, chartConfig.Ref, registryPullSecrets, ociConfig, sharedCache)
	}

	// fetch the chart from a component descriptor defined resource
	if chartConfig.FromResource != nil {
		return getChartFromResource(ctx, lsClient, contextObj, registryPullSecrets, ociConfig, sharedCache, chartConfig.FromResource)
	}

	if chartConfig.HelmChartRepo != nil {
		return getChartFromHelmChartRepo(ctx, lsClient, contextObj, chartConfig.HelmChartRepo)
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
	ociImageRef string,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache) (*chart.Chart, error) {

	resource, err := registries.NewFactory().NewHelmOCIResource(ctx, ociImageRef, registryPullSecrets, ociConfig, sharedCache)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if _, err := resource.GetBlob(ctx, &buf); err != nil {
		return nil, fmt.Errorf("unable to resolve chart from %q: %w", ociImageRef, err)
	}

	ch, err := chartloader.LoadArchive(&buf)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}
	return ch, err
}

func getChartFromResource(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	registryPullSecrets []corev1.Secret,
	ociConfig *config.OCIConfiguration,
	sharedCache cache.Cache,
	ref *helmv1alpha1.RemoteChartReference) (*chart.Chart, error) {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "getChartFromResource"})

	registryAccess, err := registries.NewFactory().NewRegistryAccessForHelm(ctx, lsClient, contextObj, registryPullSecrets, ociConfig, sharedCache, ref)
	if err != nil {
		return nil, err
	}

	cdRef := deployerlegacy.GetReferenceFromComponentDescriptorDefinition(&ref.ComponentDescriptorDefinition)
	if cdRef == nil {
		return nil, fmt.Errorf("no component descriptor reference found for %q", ref.ResourceName)
	}

	componentVersion, err := registryAccess.GetComponentVersion(ctx, cdRef)
	if err != nil {
		return nil, fmt.Errorf("unable to get component descriptor for %q: %w", cdRef.ComponentName, err)
	}

	resource, err := componentVersion.GetResource(ref.ResourceName, nil)
	if err != nil {
		logger.Error(err, "unable to find helm resource")
		return nil, fmt.Errorf("unable to find resource with name %q in component descriptor", ref.ResourceName)
	}

	var buf bytes.Buffer
	if _, err := resource.GetBlob(ctx, &buf); err != nil {
		return nil, fmt.Errorf("unable to resolve chart from resource %q: %w", ref.ResourceName, err)
	}

	ch, err := chartloader.LoadArchive(&buf)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}
	return ch, err
}

func getChartFromHelmChartRepo(ctx context.Context,
	lsClient client.Client,
	contextObj *lsv1alpha1.Context,
	repo *helmv1alpha1.HelmChartRepo) (*chart.Chart, error) {

	resource, err := registries.NewFactory().NewHelmRepoResource(ctx, repo, lsClient, contextObj)
	if err != nil {
		return nil, fmt.Errorf("unable to construct resource for chart %q with version %q from helm chart repo %q: %w",
			repo.HelmChartName, repo.HelmChartVersion, repo.HelmChartRepoUrl, err)
	}

	var buf bytes.Buffer
	if _, err := resource.GetBlob(ctx, &buf); err != nil {
		return nil, fmt.Errorf("unable to resolve chart %q with version %q from helm chart repo %q: %w",
			repo.HelmChartName, repo.HelmChartVersion, repo.HelmChartRepoUrl, err)
	}

	ch, err := chartloader.LoadArchive(&buf)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}
	return ch, err
}
