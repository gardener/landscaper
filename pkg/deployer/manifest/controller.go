// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logr.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	config manifestv1alpha2.Configuration) (deployerlib.Deployer, error) {

	return &deployer{
		log:        log,
		lsClient:   lsKubeClient,
		hostClient: hostKubeClient,
		config:     config,
	}, nil
}

type deployer struct {
	log        logr.Logger
	lsClient   client.Client
	hostClient client.Client
	config     manifestv1alpha2.Configuration
	hooks      extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	logger := d.log.WithValues("resource", types.NamespacedName{Name: di.Name, Namespace: di.Namespace}.String())
	manifest, err := New(logger, d.lsClient, d.hostClient, &d.config, di, target)
	if err != nil {
		return err
	}
	return manifest.Reconcile(ctx)
}

func (d deployer) Delete(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	logger := d.log.WithValues("resource", types.NamespacedName{Name: di.Name, Namespace: di.Namespace}.String())
	manifest, err := New(logger, d.lsClient, d.hostClient, &d.config, di, target)
	if err != nil {
		return err
	}
	return manifest.Delete(ctx)
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
