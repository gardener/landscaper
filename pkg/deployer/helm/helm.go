// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	helminstall "github.com/gardener/landscaper/apis/deployer/helm/install"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	helmv1alpha1validation "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/validation"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/helm/chartresolver"
	"github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/utils"
)

const (
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/helm"
	Name string                    = "helm.deployer.landscaper.gardener.cloud"
)

var HelmScheme = runtime.NewScheme()

func init() {
	helminstall.Install(HelmScheme)
}

// NewDeployItemBuilder creates a new deployitem builder for helm deployitems
func NewDeployItemBuilder() *utils.DeployItemBuilder {
	return utils.NewDeployItemBuilder(string(Type)).Scheme(HelmScheme)
}

// Helm is the internal representation of a DeployItem of Type Helm
type Helm struct {
	lsUncachedClient   client.Client
	lsCachedClient     client.Client
	hostUncachedClient client.Client
	hostCachedClient   client.Client
	lsRestConfig       *rest.Config

	Configuration helmv1alpha1.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.ResolvedTarget
	Context               *lsv1alpha1.Context
	ProviderConfiguration *helmv1alpha1.ProviderConfiguration
	ProviderStatus        *helmv1alpha1.ProviderStatus

	targetAccess *lib.TargetAccess
}

// New creates a new internal helm item
func New(
	lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	lsRestConfig *rest.Config,
	helmconfig helmv1alpha1.Configuration,
	item *lsv1alpha1.DeployItem,
	rt *lsv1alpha1.ResolvedTarget,
	lsCtx *lsv1alpha1.Context) (*Helm, error) {

	currOp := "InitHelmOperation"

	config := &helmv1alpha1.ProviderConfiguration{}
	helmdecoder := api.NewDecoder(HelmScheme)
	if _, _, err := helmdecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "ParseProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	if err := helmv1alpha1validation.ValidateProviderConfiguration(config); err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "ValidateProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	var status *helmv1alpha1.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &helmv1alpha1.ProviderStatus{}
		if _, _, err := helmdecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, lserrors.NewWrappedError(err,
				currOp, "ParseProviderStatus", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
		}
	}

	return &Helm{
		lsRestConfig:          lsRestConfig,
		lsUncachedClient:      lsUncachedClient,
		lsCachedClient:        lsCachedClient,
		hostUncachedClient:    hostUncachedClient,
		hostCachedClient:      hostCachedClient,
		Configuration:         helmconfig,
		DeployItem:            item,
		Target:                rt,
		Context:               lsCtx,
		ProviderConfiguration: config,
		ProviderStatus:        status,
	}, nil
}

func (h *Helm) ensureTargetAccess(ctx context.Context) (err error) {
	if h.targetAccess == nil {
		h.targetAccess, err = lib.NewTargetAccess(ctx, h.Target, h.lsUncachedClient, h.lsRestConfig)
	}
	return err
}

// Template loads the specified helm chart
// and templates it with the given values.
func (h *Helm) Template(ctx context.Context) (map[string]string, map[string]string, map[string]interface{}, *chart.Chart, lserrors.LsError) {

	currOp := "TemplateChart"

	if err := h.ensureTargetAccess(ctx); err != nil {
		return nil, nil, nil, nil, lserrors.NewWrappedError(err, currOp, "ensureTargetAccess", err.Error())
	}

	// download chart
	// todo: do caching of charts

	// resolve all registry pull secrets
	registryPullSecretRefs := lib.GetRegistryPullSecretsFromContext(h.Context)
	registryPullSecrets, err := kutil.ResolveSecrets(ctx, h.lsUncachedClient, registryPullSecretRefs)
	if err != nil {
		return nil, nil, nil, nil, lserrors.NewWrappedError(err, currOp, "ResolveSecrets", err.Error())
	}

	useChartCache := helper.HasCacheHelmChartsAnnotation(&h.DeployItem.ObjectMeta)

	ch, err := chartresolver.GetChart(ctx, &h.ProviderConfiguration.Chart, h.lsUncachedClient, h.Context,
		registryPullSecrets, h.Configuration.OCI, useChartCache)
	if err != nil {
		if h.isDownloadInfoError(err) {
			return nil, nil, nil, nil, lserrors.NewWrappedError(err, currOp, "GetHelmChart", err.Error(), lsv1alpha1.ErrorForInfoOnly)
		}
		return nil, nil, nil, nil, lserrors.NewWrappedError(err, currOp, "GetHelmChart", err.Error())
	}

	//template chart
	options := chartutil.ReleaseOptions{
		Name:      h.ProviderConfiguration.Name,
		Namespace: h.ProviderConfiguration.Namespace,
		Revision:  0,
		IsInstall: true,
	}

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(h.ProviderConfiguration.Values, &values); err != nil {
		return nil, nil, nil, nil, lserrors.NewWrappedError(
			err, currOp, "ParseHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}
	values, err = chartutil.ToRenderValues(ch, values, options, nil)
	if err != nil {
		return nil, nil, nil, nil, lserrors.NewWrappedError(
			err, currOp, "PrepareHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	// the files are only required for the manifest helm deployer
	var filesForManifestDeployer map[string]string
	crdsForManifestDeployer := map[string]string{}
	shouldUseRealHelmDeployer := ptr.Deref[bool](h.ProviderConfiguration.HelmDeployment, true)
	if !shouldUseRealHelmDeployer {
		filesForManifestDeployer, err = engine.RenderWithClient(ch, values, h.targetAccess.TargetRestConfig())
		if err != nil {
			return nil, nil, nil, nil, lserrors.NewWrappedError(
				err, currOp, "RenderHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
		}

		for _, crd := range ch.CRDObjects() {
			crdsForManifestDeployer[crd.Filename] = string(crd.File.Data[:])
		}
	}

	return filesForManifestDeployer, crdsForManifestDeployer, values, ch, nil
}

func (h *Helm) isDownloadInfoError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no chart name found") ||
		(strings.Contains(msg, "cannot get chart repository") && strings.Contains(msg, "could not find protocol handler for")) ||
		(strings.Contains(msg, "cannot download repository index for") && strings.Contains(msg, "404 Not Found"))
}
