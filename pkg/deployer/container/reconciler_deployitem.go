// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	"github.com/gardener/component-cli/ociclient/cache"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/deployer/container"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
)

const (
	cacheIdentifier = "container-deployer-controller"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logr.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	directHostClient client.Client,
	config containerv1alpha1.Configuration) (deployerlib.Deployer, error) {

	var sharedCache cache.Cache
	if config.OCI != nil && config.OCI.Cache != nil {
		var err error
		sharedCache, err = cache.NewCache(log, utils.ToOCICacheOptions(config.OCI.Cache, cacheIdentifier)...)
		if err != nil {
			return nil, err
		}
	}

	return &deployer{
		log:              log,
		lsClient:         lsKubeClient,
		hostClient:       hostKubeClient,
		directHostClient: directHostClient,
		config:           config,
		sharedCache:      sharedCache,
	}, nil
}

type deployer struct {
	log              logr.Logger
	lsClient         client.Client
	hostClient       client.Client
	directHostClient client.Client
	config           containerv1alpha1.Configuration
	sharedCache      cache.Cache
	hooks            extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, di *lsv1alpha1.DeployItem, _ *lsv1alpha1.Target) error {
	logger := d.log.WithValues("resource", types.NamespacedName{Name: di.Name, Namespace: di.Namespace})
	containerOp, err := New(logger, d.lsClient, d.hostClient, d.directHostClient, d.config, di, d.sharedCache)
	if err != nil {
		return err
	}
	return containerOp.Reconcile(ctx, container.OperationReconcile)
}

func (d deployer) Delete(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	logger := d.log.WithValues("resource", types.NamespacedName{Name: di.Name, Namespace: di.Namespace})
	containerOp, err := New(logger, d.lsClient, d.hostClient, d.directHostClient, d.config, di, d.sharedCache)
	if err != nil {
		return err
	}
	return containerOp.Delete(ctx)
}

func (d *deployer) ForceReconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	return d.Reconcile(ctx, di, target)
}

func (d *deployer) Abort(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	d.log.Info("abort is not yet implemented")
	return nil
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}
