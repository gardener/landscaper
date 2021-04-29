// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

const (
	cacheIdentifier = "helm-deployer-controller"
)

// NewDeployer creates a new deployer that reconciles deploy items of type helm.
func NewDeployer(log logr.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	config helmv1alpha1.Configuration) (deployerlib.Deployer, error) {

	componentRegistryMgr, err := componentsregistry.SetupManagerFromConfig(log, config.OCI, cacheIdentifier)
	if err != nil {
		return nil, err
	}

	return &deployer{
		log:                   log,
		lsClient:              lsKubeClient,
		hostClient:            hostKubeClient,
		config:                config,
		componentsRegistryMgr: componentRegistryMgr,
	}, nil
}

type deployer struct {
	log                   logr.Logger
	lsClient              client.Client
	hostClient            client.Client
	config                helmv1alpha1.Configuration
	componentsRegistryMgr *componentsregistry.Manager
}

func (d *deployer) Reconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	helm, err := New(d.log, d.config, d.lsClient, di, target, d.componentsRegistryMgr)
	if err != nil {
		return err
	}
	di.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	files, values, err := helm.Template(ctx)
	if err != nil {
		return err
	}

	exports, err := helm.constructExportsFromValues(values)
	if err != nil {
		di.Status.LastError = lsv1alpha1helper.UpdatedError(di.Status.LastError,
			"ConstructExportFromValues", "", err.Error())
		return err
	}
	return helm.ApplyFiles(ctx, files, exports)
}

func (d *deployer) Delete(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	helm, err := New(d.log, d.config, d.lsClient, di, target, d.componentsRegistryMgr)
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
