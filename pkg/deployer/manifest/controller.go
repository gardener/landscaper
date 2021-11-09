// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	cr "github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logr.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	config manifestv1alpha2.Configuration) (deployerlib.Deployer, error) {

	dep := &deployer{
		log:        log,
		lsClient:   lsKubeClient,
		hostClient: hostKubeClient,
		config:     config,
		hooks:      extension.ReconcileExtensionHooks{},
	}
	dep.hooks.RegisterHookSetup(cr.ContinuousReconcileExtensionSetup(dep.NextReconcile))
	return dep, nil
}

type deployer struct {
	log        logr.Logger
	lsClient   client.Client
	hostClient client.Client
	config     manifestv1alpha2.Configuration
	hooks      extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	logger := d.log.WithValues("resource", types.NamespacedName{Name: di.Name, Namespace: di.Namespace}.String())
	manifest, err := New(logger, d.lsClient, d.hostClient, &d.config, di, target)
	if err != nil {
		return err
	}
	return manifest.Reconcile(ctx)
}

func (d deployer) Delete(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	logger := d.log.WithValues("resource", types.NamespacedName{Name: di.Name, Namespace: di.Namespace}.String())
	manifest, err := New(logger, d.lsClient, d.hostClient, &d.config, di, target)
	if err != nil {
		return err
	}
	return manifest.Delete(ctx)
}

func (d *deployer) ForceReconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if err := d.Reconcile(ctx, lsCtx, di, target); err != nil {
		return err
	}
	delete(di.Annotations, lsv1alpha1.OperationAnnotation)
	return nil
}

func (d *deployer) Abort(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	d.log.Info("abort is not yet implemented")
	return nil
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}

func (d *deployer) NextReconcile(ctx context.Context, last time.Time, di *lsv1alpha1.DeployItem) (*time.Time, error) {
	manifest, err := New(d.log, d.lsClient, d.hostClient, &d.config, di, nil)
	if err != nil {
		return nil, err
	}
	if crval.ContinuousReconcileSpecIsEmpty(manifest.ProviderConfiguration.ContinuousReconcile) {
		// no continuous reconciliation configured
		return nil, nil
	}
	schedule, err := cr.Schedule(manifest.ProviderConfiguration.ContinuousReconcile)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(last)
	return &next, nil
}
