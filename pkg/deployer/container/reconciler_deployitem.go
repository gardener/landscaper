// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"time"

	"github.com/gardener/landscaper/pkg/utils/cd_facade"

	"github.com/gardener/landscaper/apis/deployer/container"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/deployer/lib/extension"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	cr "github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"
)

const (
	cacheIdentifier = "container-deployer-controller"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logging.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	directHostClient client.Client,
	config containerv1alpha1.Configuration) (*deployer, error) {

	var sharedCache cd_facade.Cache
	if config.OCI != nil && config.OCI.Cache != nil {
		var err error
		sharedCache, err = cd_facade.NewCache(log.Logr(), config.OCI.Cache, cacheIdentifier)
		if err != nil {
			return nil, err
		}
	}

	dep := &deployer{
		log:              log,
		lsClient:         lsKubeClient,
		hostClient:       hostKubeClient,
		directHostClient: directHostClient,
		config:           config,
		sharedCache:      sharedCache,
		hooks:            extension.ReconcileExtensionHooks{},
	}
	dep.hooks.RegisterHookSetup(cr.ContinuousReconcileExtensionSetup(dep.NextReconcile))
	return dep, nil
}

type deployer struct {
	log              logging.Logger
	lsClient         client.Client
	hostClient       client.Client
	directHostClient client.Client
	config           containerv1alpha1.Configuration
	sharedCache      cd_facade.Cache
	hooks            extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	containerOp, err := New(d.lsClient, d.hostClient, d.directHostClient, d.config, di, lsCtx, d.sharedCache, rt)
	if err != nil {
		return err
	}
	ctx = logging.NewContext(ctx, d.log)
	return containerOp.Reconcile(ctx, container.OperationReconcile)
}

func (d deployer) Delete(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, rt *lsv1alpha1.ResolvedTarget) error {
	containerOp, err := New(d.lsClient, d.hostClient, d.directHostClient, d.config, di, lsCtx, d.sharedCache, rt)
	if err != nil {
		return err
	}
	ctx = logging.NewContext(ctx, d.log)
	return containerOp.Delete(ctx)
}

func (d *deployer) Abort(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, _ *lsv1alpha1.ResolvedTarget) error {
	d.log.Info("abort is not yet implemented")
	return nil
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}

func (d *deployer) NextReconcile(ctx context.Context, last time.Time, di *lsv1alpha1.DeployItem) (*time.Time, error) {
	// TODO: parse provider configuration directly and do not init the container helper struct
	containerOp, err := New(d.lsClient, d.hostClient, d.directHostClient, d.config, di, nil, d.sharedCache, nil)
	if err != nil {
		return nil, err
	}
	if crval.ContinuousReconcileSpecIsEmpty(containerOp.ProviderConfiguration.ContinuousReconcile) {
		// no continuous reconciliation configured
		return nil, nil
	}
	schedule, err := cr.Schedule(containerOp.ProviderConfiguration.ContinuousReconcile)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(last)
	return &next, nil
}
