// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"time"

	"github.com/gardener/landscaper/pkg/components/registries"

	"github.com/gardener/component-cli/ociclient/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	cnudieutils "github.com/gardener/landscaper/pkg/components/cnudie/utils"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	cr "github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
)

const (
	cacheIdentifier = "helm-deployer-controller"
)

const (
	TimeoutCheckpointHelmStartReconcile            = "helm deployer: start reconcile"
	TimeoutCheckpointHelmStartProgressing          = "helm deployer: start progressing"
	TimeoutCheckpointHelmStartApplyFiles           = "helm deployer: start apply files"
	TimeoutCheckpointHelmStartDelete               = "helm deployer: start delete"
	TimeoutCheckpointHelmStartDeleting             = "helm deployer: start deleting"
	TimeoutCheckpointHelmBeforeReadinessCheck      = "helm deployer: before readiness check"
	TimeoutCheckpointHelmBeforeReadingExportValues = "helm deployer: before reading export values"
	TimeoutCheckpointHelmStartApplyManifests       = "helm deployer: start apply manifests"
	TimeoutCheckpointHelmStartCreateManifests      = "helm deployer: start create manifests"
	TimeoutCheckpointHelmDefaultReadinessChecks    = "helm deployer: default readiness checks"
	TimeoutCheckpointHelmCustomReadinessChecks     = "helm deployer: custom readiness checks"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	log logging.Logger,
	config helmv1alpha1.Configuration) (deployerlib.Deployer, error) {

	var sharedCache cache.Cache
	if config.OCI != nil && config.OCI.Cache != nil {
		var err error
		sharedCache, err = cache.NewCache(log.Logr(), cnudieutils.ToOCICacheOptions(config.OCI.Cache, cacheIdentifier)...)
		if err != nil {
			return nil, err
		}
	}

	registries.SetOCMLibraryMode(config.UseOCMLib)

	dep := &deployer{
		lsUncachedClient:   lsUncachedClient,
		lsCachedClient:     lsCachedClient,
		hostUncachedClient: hostUncachedClient,
		hostCachedClient:   hostCachedClient,
		log:                log,
		config:             config,
		sharedCache:        sharedCache,
		hooks:              extension.ReconcileExtensionHooks{},
	}
	dep.hooks.RegisterHookSetup(cr.ContinuousReconcileExtensionSetup(dep.NextReconcile))
	return dep, nil
}

type deployer struct {
	lsUncachedClient   client.Client
	lsCachedClient     client.Client
	hostUncachedClient client.Client
	hostCachedClient   client.Client

	log         logging.Logger
	config      helmv1alpha1.Configuration
	sharedCache cache.Cache
	hooks       extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	if _, err := timeout.TimeoutExceeded(ctx, di, TimeoutCheckpointHelmStartReconcile); err != nil {
		return err
	}

	helm, err := New(d.lsUncachedClient, d.lsCachedClient, d.hostUncachedClient, d.hostCachedClient, d.config, di, rt, lsCtx, d.sharedCache)
	if err != nil {
		err = lserrors.NewWrappedError(err, "Reconcile", "newRootLogger", err.Error())
		return err
	}

	if _, err := timeout.TimeoutExceeded(ctx, di, TimeoutCheckpointHelmStartProgressing); err != nil {
		return err
	}

	di.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing

	// filesForManifestDeployer and crdsForManifestDeployer are only required for the helm manifest deployer and otherwise empty
	filesForManifestDeployer, crdsForManifestDeployer, values, ch, err := helm.Template(ctx)
	if err != nil {
		err = lserrors.NewWrappedError(err, "Reconcile", "Template", err.Error())
		return err
	}

	exports, err := helm.constructExportsFromValues(values)
	if err != nil {
		err = lserrors.NewWrappedError(err, "Reconcile", "ConstructExportFromValues", err.Error())
		return err
	}

	if _, err := timeout.TimeoutExceeded(ctx, di, TimeoutCheckpointHelmStartApplyFiles); err != nil {
		return err
	}

	return helm.ApplyFiles(ctx, filesForManifestDeployer, crdsForManifestDeployer, exports, ch)
}

func (d *deployer) Delete(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	if _, err := timeout.TimeoutExceeded(ctx, di, TimeoutCheckpointHelmStartDelete); err != nil {
		return err
	}

	helm, err := New(d.lsUncachedClient, d.lsCachedClient, d.hostUncachedClient, d.hostCachedClient, d.config, di, rt, lsCtx, d.sharedCache)
	if err != nil {
		return err
	}

	if _, err := timeout.TimeoutExceeded(ctx, di, TimeoutCheckpointHelmStartDeleting); err != nil {
		return err
	}

	return helm.DeleteFiles(ctx)
}

func (d *deployer) Abort(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	d.log.Info("abort is not yet implemented")
	return nil
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}

func (d *deployer) NextReconcile(ctx context.Context, last time.Time, di *lsv1alpha1.DeployItem) (*time.Time, error) {
	// todo: directly parse deploy items
	helm, err := New(d.lsUncachedClient, d.lsCachedClient, d.hostUncachedClient, d.hostCachedClient, d.config, di, nil, nil, d.sharedCache)
	if err != nil {
		return nil, err
	}
	if crval.ContinuousReconcileSpecIsEmpty(helm.ProviderConfiguration.ContinuousReconcile) {
		// no continuous reconciliation configured
		return nil, nil
	}
	schedule, err := cr.Schedule(helm.ProviderConfiguration.ContinuousReconcile)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(last)
	return &next, nil
}
