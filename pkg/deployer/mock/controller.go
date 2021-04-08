// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// NewController creates a new deploy item controller that reconciles deploy items of type mock.
func NewController(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme, config *mockv1alpha1.Configuration) (reconcile.Reconciler, error) {
	return &controller{
		log:    log,
		client: kubeClient,
		scheme: scheme,
		config: config,
	}, nil
}

type controller struct {
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme
	config *mockv1alpha1.Configuration
}

func (a *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := a.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile")

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.client.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			a.log.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if deployItem.Spec.Type != Type {
		return reconcile.Result{}, nil
	}

	var target *lsv1alpha1.Target
	if deployItem.Spec.Target != nil {
		target = &lsv1alpha1.Target{}
		if err := a.client.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to get target for deploy item: %w", err)
		}
		if len(a.config.TargetSelector) != 0 {
			matched, err := targetselector.Match(target, a.config.TargetSelector)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("unable to match target selector: %w", err)
			}
			if !matched {
				logger.V(5).Info("The deploy item's target does not match the given target selector")
				return reconcile.Result{}, nil
			}
		}
	}

	logger.Info("reconcile mock deploy item")

	err := deployerlib.HandleAnnotationsAndGeneration(ctx, logger, a.client, deployItem)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := a.reconcile(ctx, deployItem); err != nil {
		a.log.Error(err, "unable to reconcile deploy item")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *controller) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem) error {
	if !deployItem.DeletionTimestamp.IsZero() {
		if err := a.ensureDeletion(ctx, deployItem); err != nil {
			return err
		}
		controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		return a.client.Update(ctx, deployItem)
	} else if !kubernetesutil.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		return a.client.Update(ctx, deployItem)
	}

	config, err := a.getConfig(ctx, deployItem)
	if err != nil {
		return err
	}

	if err := a.ensureExport(ctx, deployItem, config); err != nil {
		return err
	}

	deployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded

	if config.Phase != nil {
		deployItem.Status.Phase = *config.Phase
	}

	if config.ProviderStatus != nil {
		deployItem.Status.ProviderStatus = config.ProviderStatus
	}

	return a.client.Status().Update(ctx, deployItem)
}

func (a *controller) ensureDeletion(ctx context.Context, item *lsv1alpha1.DeployItem) error {
	if item.Status.ExportReference == nil {
		return nil
	}
	secret := &corev1.Secret{}
	secret.Name = item.Status.ExportReference.Name
	secret.Namespace = item.Status.ExportReference.Namespace

	if err := a.client.Delete(ctx, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (a *controller) ensureExport(ctx context.Context, item *lsv1alpha1.DeployItem, config *mockv1alpha1.ProviderConfiguration) error {
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

	_, err := kubernetesutil.CreateOrUpdate(ctx, a.client, secret, func() error {
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

	return a.client.Status().Update(ctx, item)
}

func (a *controller) getConfig(ctx context.Context, item *lsv1alpha1.DeployItem) (*mockv1alpha1.ProviderConfiguration, error) {
	config := &mockv1alpha1.ProviderConfiguration{}
	if _, _, err := Decoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		a.log.Error(err, "unable to unmarshal config")
		item.Status.Conditions = lsv1alpha1helper.CreateOrUpdateConditions(item.Status.Conditions, lsv1alpha1.DeployItemValidationCondition, lsv1alpha1.ConditionFalse,
			"FailedUnmarshal", err.Error())
		_ = a.client.Status().Update(ctx, item)
		return nil, err
	}
	item.Status.Conditions = lsv1alpha1helper.CreateOrUpdateConditions(item.Status.Conditions, lsv1alpha1.DeployItemValidationCondition, lsv1alpha1.ConditionTrue,
		"SuccessfullValidation", "Successfully validated configuration")
	_ = a.client.Status().Update(ctx, item)
	return config, nil
}
