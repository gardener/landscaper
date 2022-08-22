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

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/helm/helmchartrepo"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"helm.sh/helm/v3/pkg/chart"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

// NoChartDefinedError is the error that is returned if no Helm chart was provided
var NoChartDefinedError = errors.New("no chart was provided")

// GetChart resolves the chart based on a chart access configuration.
func GetChart(ctx context.Context, ociClient ociclient.Client,
	helmChartRepoClient *helmchartrepo.HelmChartRepoClient, chartConfig *helmv1alpha1.Chart) (*chart.Chart, error) {

	if chartConfig.Archive != nil {
		return getChartFromArchive(chartConfig.Archive)
	}

	if len(chartConfig.Ref) != 0 {
		return getChartFromOCIRef(ctx, ociClient, chartConfig.Ref)
	}

	// fetch the chart from a component descriptor defined resource
	if chartConfig.FromResource != nil {
		return getChartFromResource(ctx, ociClient, helmChartRepoClient, chartConfig.FromResource)
	}

	if chartConfig.HelmChartRepo != nil {
		return getChartFromHelmChartRepo(ctx, helmChartRepoClient, chartConfig.HelmChartRepo)
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

func getChartFromOCIRef(ctx context.Context, ociClient ociclient.Client, ref string) (*chart.Chart, error) {
	ociAccess := cdv2.NewOCIRegistryAccess(ref)
	access, err := cdv2.NewUnstructured(ociAccess)
	if err != nil {
		return nil, fmt.Errorf("unable to construct ociClient registry access for %q: %w", ref, err)
	}

	res := cdv2.Resource{
		// only the type is needed other attributes can be ommitted.
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Type: HelmChartResourceType,
		},
		Relation: cdv2.ExternalRelation,
		Access:   &access,
	}
	var buf bytes.Buffer
	if _, err := NewHelmResolver(ociClient).Resolve(ctx, res, &buf); err != nil {
		return nil, fmt.Errorf("unable to resolve chart from %q: %w", ref, err)
	}

	ch, err := chartloader.LoadArchive(&buf)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}
	return ch, err
}

func getChartFromResource(ctx context.Context, ociClient ociclient.Client,
	helmChartRepoClient *helmchartrepo.HelmChartRepoClient, ref *helmv1alpha1.RemoteChartReference) (*chart.Chart, error) {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "getChartFromResource"})

	// we also have to add a custom resolver for the "ociImage" resolver as we have to implement the
	// helm specific ociClient manifest structure
	compResolver, err := componentsregistry.NewOCIRegistryWithOCIClient(logger, ociClient, ref.Inline)
	if err != nil {
		return nil, fmt.Errorf("unable to build component resolver: %w", err)
	}

	cdRef := installations.GetReferenceFromComponentDescriptorDefinition(&ref.ComponentDescriptorDefinition)
	if cdRef == nil {
		return nil, fmt.Errorf("no component descriptor reference found for %q", ref.ResourceName)
	}

	cd, blobResolver, err := compResolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to get component descriptor for %q: %w", cdRef.ComponentName, err)
	}

	// add customized helm resolver
	blobResolver, err = ctf.AggregateBlobResolvers(blobResolver, NewHelmResolver(ociClient))
	if err != nil {
		return nil, fmt.Errorf("unable to add helm chart specific chart resolver: %w", err)
	}

	blobResolver, err = ctf.AggregateBlobResolvers(blobResolver, helmchartrepo.NewHelmChartRepoResolver(helmChartRepoClient))
	if err != nil {
		return nil, fmt.Errorf("unable to add helm chart repo specific chart resolver: %w", err)
	}

	resources, err := cd.GetResourcesByName(ref.ResourceName)
	if err != nil {
		logger.Error(err, "unable to find helm resource")
		return nil, fmt.Errorf("unable to find resource with name %q in component descriptor", ref.ResourceName)
	}
	if len(resources) != 1 {
		return nil, fmt.Errorf("resource with name %q cannot be uniquly identified", ref.ResourceName)
	}
	res := resources[0]

	var buf bytes.Buffer
	if _, err := blobResolver.Resolve(ctx, res, &buf); err != nil {
		return nil, fmt.Errorf("unable to resolve chart from resource %q: %w", ref.ResourceName, err)
	}

	ch, err := chartloader.LoadArchive(&buf)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}
	return ch, err
}

func getChartFromHelmChartRepo(ctx context.Context, helmChartRepoClient *helmchartrepo.HelmChartRepoClient,
	ref *helmv1alpha1.HelmChartRepo) (*chart.Chart, error) {
	resolver := helmchartrepo.NewHelmChartRepoResolverAsHelmChartRepoResolver(helmChartRepoClient)
	var buf bytes.Buffer

	if _, err := resolver.ResolveHelmChart(ctx, ref, &buf); err != nil {
		return nil, fmt.Errorf("unable to resolve chart %q with version %q from helm chart repo %q: %w",
			ref.HelmChartName, ref.HelmChartVersion, ref.HelmChartRepoUrl, err)
	}

	ch, err := chartloader.LoadArchive(&buf)
	if err != nil {
		return nil, fmt.Errorf("unable to load chart from archive: %w", err)
	}
	return ch, err
}
