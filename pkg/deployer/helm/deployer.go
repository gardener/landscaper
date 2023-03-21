// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"time"

	"github.com/gardener/landscaper/pkg/utils/cd_facade"

	"github.com/gardener/component-cli/ociclient/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	cr "github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

const (
	cacheIdentifier = "helm-deployer-controller"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logging.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	config helmv1alpha1.Configuration) (deployerlib.Deployer, error) {

	var sharedCache cache.Cache
	if config.OCI != nil && config.OCI.Cache != nil {
		var err error
		sharedCache, err = cd_facade.NewCache(log.Logr(), config.OCI.Cache, cacheIdentifier)
		if err != nil {
			return nil, err
		}
	}

	dep := &deployer{
		log:         log,
		lsClient:    lsKubeClient,
		hostClient:  hostKubeClient,
		config:      config,
		sharedCache: sharedCache,
		hooks:       extension.ReconcileExtensionHooks{},
	}
	dep.hooks.RegisterHookSetup(cr.ContinuousReconcileExtensionSetup(dep.NextReconcile))
	return dep, nil
}

type deployer struct {
	log         logging.Logger
	lsClient    client.Client
	hostClient  client.Client
	config      helmv1alpha1.Configuration
	sharedCache cache.Cache
	hooks       extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	helm, err := New(d.config, d.lsClient, d.hostClient, di, rt, lsCtx, d.sharedCache)
	if err != nil {
		err = lserrors.NewWrappedError(err, "Reconcile", "New", err.Error())
		return err
	}
	di.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing

	files, crds, values, ch, err := helm.Template(ctx, d.lsClient)
	if err != nil {
		err = lserrors.NewWrappedError(err, "Reconcile", "Template", err.Error())
		return err
	}

	exports, err := helm.constructExportsFromValues(values)
	if err != nil {
		err = lserrors.NewWrappedError(err, "Reconcile", "ConstructExportFromValues", err.Error())
		return err
	}
	return helm.ApplyFiles(ctx, files, crds, exports, ch)
}

func (d *deployer) Delete(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	helm, err := New(d.config, d.lsClient, d.hostClient, di, rt, lsCtx, d.sharedCache)
	if err != nil {
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
	helm, err := New(d.config, d.lsClient, d.hostClient, di, nil, nil, d.sharedCache)
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
