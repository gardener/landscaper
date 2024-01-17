// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"context"
	"time"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	kubernetesutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	cr "github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

// NewDeployer creates a new deployer that reconciles deploy items of type mock.
func NewDeployer(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	log logging.Logger,
	config mockv1alpha1.Configuration) (deployerlib.Deployer, error) {

	dep := &deployer{
		lsUncachedClient:   lsUncachedClient,
		lsCachedClient:     lsCachedClient,
		hostUncachedClient: hostUncachedClient,
		hostCachedClient:   hostCachedClient,
		log:                log,
		config:             config,
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

	log    logging.Logger
	config mockv1alpha1.Configuration
	hooks  extension.ReconcileExtensionHooks
}

func (d *deployer) Reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, _ *lsv1alpha1.ResolvedTarget) error {
	config, err := d.getConfig(ctx, di)
	if err != nil {
		return err
	}

	if err := d.ensureExport(ctx, di, config); err != nil {
		return err
	}

	if config.InitialPhase != nil {
		if len(di.Status.Phase) == 0 || di.Status.Phase == lsv1alpha1.DeployItemPhases.Init {
			di.Status.Phase = *config.InitialPhase
		}
	} else {
		di.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
	}

	if config.Phase != nil {
		di.Status.Phase = *config.Phase
	}

	if config.ProviderStatus != nil {
		di.Status.ProviderStatus = config.ProviderStatus
	}

	return d.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000059, di)
}

func (d *deployer) Delete(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, _ *lsv1alpha1.ResolvedTarget) error {
	return d.ensureDeletion(ctx, di)
}

func (d *deployer) Abort(ctx context.Context, lsCtx *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, _ *lsv1alpha1.ResolvedTarget) error {
	d.log.Info("abort is not yet implemented")
	return nil
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}

func (d *deployer) ensureDeletion(ctx context.Context, item *lsv1alpha1.DeployItem) error {
	if item.Status.ExportReference == nil {
		return nil
	}
	secret := &corev1.Secret{}
	secret.Name = item.Status.ExportReference.Name
	secret.Namespace = item.Status.ExportReference.Namespace

	if err := d.lsUncachedClient.Delete(ctx, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (d *deployer) ensureExport(ctx context.Context, item *lsv1alpha1.DeployItem, config *mockv1alpha1.ProviderConfiguration) error {
	if config.Export == nil {
		return nil
	}

	secret := &corev1.Secret{}
	secret.GenerateName = "mock-export-"
	secret.Namespace = item.Namespace
	if item.Status.ExportReference != nil {
		secret.Name = item.Status.ExportReference.Name
		secret.Namespace = item.Status.ExportReference.Namespace
	}

	_, err := kubernetesutil.CreateOrUpdate(ctx, d.lsUncachedClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: *config.Export,
		}
		return controllerutil.SetOwnerReference(item, secret, api.LandscaperScheme)
	})
	if err != nil {
		return err
	}

	item.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	return d.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000061, item)
}

func (d *deployer) getConfig(ctx context.Context, item *lsv1alpha1.DeployItem) (*mockv1alpha1.ProviderConfiguration, error) {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "getConfig"})

	config := &mockv1alpha1.ProviderConfiguration{}
	if _, _, err := Decoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		logger.Error(err, "unable to unmarshal config")
		item.Status.Conditions = lsv1alpha1helper.CreateOrUpdateConditions(item.Status.Conditions, lsv1alpha1.DeployItemValidationCondition, lsv1alpha1.ConditionFalse,
			"FailedUnmarshal", err.Error())
		_ = d.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000053, item)
		return nil, err
	}
	item.Status.Conditions = lsv1alpha1helper.CreateOrUpdateConditions(item.Status.Conditions, lsv1alpha1.DeployItemValidationCondition, lsv1alpha1.ConditionTrue,
		"SuccessfullValidation", "Successfully validated configuration")
	_ = d.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000054, item)
	return config, nil
}

func (d *deployer) NextReconcile(ctx context.Context, last time.Time, di *lsv1alpha1.DeployItem) (*time.Time, error) {
	config, err := d.getConfig(ctx, di)
	if err != nil {
		return nil, err
	}
	if crval.ContinuousReconcileSpecIsEmpty(config.ContinuousReconcile) {
		// no continuous reconciliation configured
		return nil, nil
	}
	schedule, err := cr.Schedule(config.ContinuousReconcile)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(last)
	return &next, nil
}

func (d *deployer) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(d.lsUncachedClient)
}
