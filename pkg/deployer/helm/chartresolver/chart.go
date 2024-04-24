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

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/helm/loader"

	helmid "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/runtime"
	"sigs.k8s.io/yaml"

	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/gardener/landscaper/pkg/deployer/lib"

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
	sharedCache cache.Cache,
	useChartCache bool) (*chart.Chart, error) {

	if chartConfig.Archive != nil {
		return getChartFromArchive(chartConfig.Archive)
	}

	var chart *chart.Chart
	var err error

	if useChartCache {
		chart, err = GetHelmChartCache(MaxSizeInByteDefault, RemoveOutdatedDurationDefault).getChart(chartConfig.Ref,
			chartConfig.HelmChartRepo, chartConfig.ResourceRef)
		if err != nil {
			return nil, err
		}
	}

	if chart == nil {
		if len(chartConfig.Ref) != 0 {
			chart, err = getChartFromOCIRef(ctx, contextObj, chartConfig.Ref, registryPullSecrets, ociConfig, sharedCache)
		} else if chartConfig.HelmChartRepo != nil {
			chart, err = getChartFromHelmChartRepo(ctx, lsClient, contextObj, chartConfig.HelmChartRepo)
		} else if chartConfig.FromResource != nil {
			chart, err = nil, errors.New("chart.fromResource is no longer supported")
		} else if chartConfig.ResourceRef != "" {
			chart, err = getChartFromResourceRef(ctx, chartConfig.ResourceRef, contextObj, lsClient)
		} else {
			chart, err = nil, NoChartDefinedError
		}

		if err != nil {
			return nil, err
		}
	}

	if useChartCache {
		if err = GetHelmChartCache(MaxSizeInByteDefault, RemoveOutdatedDurationDefault).addOrUpdateChart(ctx,
			chartConfig.Ref, chartConfig.HelmChartRepo, chartConfig.ResourceRef, chart); err != nil {
			return nil, err
		}
	}

	return chart, nil
}

func getChartFromResourceRef(ctx context.Context, resourceRef string, lsCtx *lsv1alpha1.Context,
	lsClient client.Client) (_ *chart.Chart, err error) {

	op := "getChartFromResourceRef"

	octx := ocm.New(datacontext.MODE_EXTENDED)

	if lsCtx == nil {
		return nil, lserrors.NewError(op, "NoContext", "landscaper context cannot be nil", lsv1alpha1.ErrorForInfoOnly,
			lsv1alpha1.ErrorConfigurationProblem)
	}

	if lsCtx == nil || lsCtx.RepositoryContext == nil || lsCtx.RepositoryContext.Raw == nil {
		msg := fmt.Sprintf("landscaper context %s/%s does not specify a repository context but has"+
			" to specify a repository context to resolve resource from an ocm reference", lsCtx.Namespace, lsCtx.Name)
		return nil, lserrors.NewError(op, "NoContext", msg, lsv1alpha1.ErrorForInfoOnly, lsv1alpha1.ErrorConfigurationProblem)
	}

	// Credential Handling
	// resolve all credentials from registry pull secrets
	registryPullSecretRefs := lib.GetRegistryPullSecretsFromContext(lsCtx)
	registryPullSecrets, err := kutil.ResolveSecrets(ctx, lsClient, registryPullSecretRefs)
	if err != nil {
		return nil, fmt.Errorf("error resolving secrets: %w", err)
	}

	err = ocmlib.AddSecretCredsToCredContext(registryPullSecrets, octx)
	if err != nil {
		return nil, err
	}

	// resolve all credentials for helm chart repositories
	if lsCtx != nil && lsCtx.Configurations != nil {
		if rawAuths, ok := lsCtx.Configurations[helmv1alpha1.HelmChartRepoCredentialsKey]; ok {
			repoCredentials := helmv1alpha1.HelmChartRepoCredentials{}
			err := yaml.Unmarshal(rawAuths.RawMessage, &repoCredentials)
			if err != nil {
				return nil, lserrors.NewWrappedError(err, "NewHelmChartRepoClient", "ParsingAuths", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
			}

			for _, a := range repoCredentials.Auths {
				id := helmid.GetConsumerId(a.URL, "")
				octx.CredentialsContext().SetCredentialsForConsumer(id, ocmlib.NewHelmCredentialSource(lsClient, a, lsCtx.Namespace))
			}
		}
	}

	// Business Logic
	key, err := base64.StdEncoding.DecodeString(resourceRef)
	if err != nil {
		return nil, err
	}

	// TODO: implement a MUX so this could deal with multiple kinds of requests
	globalId := model.GlobalResourceIdentity{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(key, &globalId)
	if err != nil {
		return nil, err
	}

	spec, err := octx.RepositorySpecForConfig(lsCtx.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
	if err != nil {
		return nil, err
	}

	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)

	repo, err := spec.Repository(octx, nil)
	if err != nil {
		return nil, err
	}
	finalize.Close(repo)

	compvers, err := repo.LookupComponentVersion(globalId.ComponentIdentity.Name, globalId.ComponentIdentity.Version)
	if err != nil {
		return nil, err
	}
	finalize.Close(compvers)

	res, err := compvers.GetResource(globalId.ResourceIdentity)
	if err != nil {
		return nil, err
	}

	fs := memoryfs.New()
	path, err := download.DownloadResource(octx, res, filepath.Join("/", "chart"), download.WithFileSystem(fs))
	if err != nil {
		return nil, err
	}
	chart, err := loader.Load(path, fs)
	if err != nil {
		return nil, err
	}
	return chart, nil
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
