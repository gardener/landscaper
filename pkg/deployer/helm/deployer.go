// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"time"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	cr "github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

const (
	cacheIdentifier = "helm-deployer-controller"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logr.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	config helmv1alpha1.Configuration) (deployerlib.Deployer, error) {

	var sharedCache cache.Cache
	if config.OCI != nil && config.OCI.Cache != nil {
		var err error
		sharedCache, err = cache.NewCache(log, utils.ToOCICacheOptions(config.OCI.Cache, cacheIdentifier)...)
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
	log         logr.Logger
	lsClient    client.Client
	hostClient  client.Client
	config      helmv1alpha1.Configuration
	sharedCache cache.Cache
	hooks       extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	helm, err := New(d.log, d.config, d.lsClient, d.hostClient, di, target, d.sharedCache)
	if err != nil {
		return err
	}
	di.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	files, crds, values, err := helm.Template(ctx)
	if err != nil {
		return err
	}

	exports, err := helm.constructExportsFromValues(values)
	if err != nil {
		di.Status.LastError = lserrors.UpdatedError(di.Status.LastError,
			"ConstructExportFromValues", "", err.Error())
		return err
	}
	return helm.ApplyFiles(ctx, files, crds, exports)
}

func (d *deployer) Delete(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	helm, err := New(d.log, d.config, d.lsClient, d.hostClient, di, target, d.sharedCache)
	if err != nil {
		return err
	}

	return helm.DeleteFiles(ctx)
}

func (d *deployer) ForceReconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if err := d.Reconcile(ctx, di, target); err != nil {
		return err
	}
	delete(di.Annotations, lsv1alpha1.OperationAnnotation)
	return nil
}

func (d *deployer) Abort(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	d.log.Info("abort is not yet implemented")
	return nil
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}

func (d *deployer) NextReconcile(ctx context.Context, last time.Time, di *lsv1alpha1.DeployItem) (*time.Time, error) {
	helm, err := New(d.log, d.config, d.lsClient, d.hostClient, di, nil, d.sharedCache)
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
