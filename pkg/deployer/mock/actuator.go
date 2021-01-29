// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func NewActuator() (reconcile.Reconciler, error) {
	return &actuator{}, nil
}

type actuator struct {
	log    logr.Logger
	c      client.Client
	scheme *runtime.Scheme
}

var _ inject.Client = &actuator{}

var _ inject.Logger = &actuator{}

var _ inject.Scheme = &actuator{}

// InjectClients injects the current kubernetes registryClient into the actuator
func (a *actuator) InjectClient(c client.Client) error {
	a.c = c
	return nil
}

// InjectLogger injects a logging instance into the actuator
func (a *actuator) InjectLogger(log logr.Logger) error {
	a.log = log
	return nil
}

// InjectScheme injects the current scheme into the actuator
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

func (a *actuator) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	a.log.Info("reconcile", "resource", req.NamespacedName)

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.c.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			a.log.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if deployItem.Spec.Type != Type {
		return reconcile.Result{}, nil
	}

	if deployItem.Status.ObservedGeneration == deployItem.Generation {
		return reconcile.Result{}, nil
	}

	if err := a.reconcile(ctx, deployItem); err != nil {
		a.log.Error(err, "unable to reconcile deploy item")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem) error {
	if !deployItem.DeletionTimestamp.IsZero() {
		if err := a.ensureDeletion(ctx, deployItem); err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		return a.c.Update(ctx, deployItem)
	} else if !kubernetesutil.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		return a.c.Update(ctx, deployItem)
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

	deployItem.Status.ObservedGeneration = deployItem.Generation
	return a.c.Status().Update(ctx, deployItem)
}

func (a *actuator) ensureDeletion(ctx context.Context, item *lsv1alpha1.DeployItem) error {
	if item.Status.ExportReference == nil {
		return nil
	}
	secret := &corev1.Secret{}
	secret.Name = item.Status.ExportReference.Name
	secret.Namespace = item.Status.ExportReference.Namespace

	if err := a.c.Delete(ctx, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (a *actuator) ensureExport(ctx context.Context, item *lsv1alpha1.DeployItem, config *mockv1alpha1.ProviderConfiguration) error {
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

	_, err := kubernetesutil.CreateOrUpdate(ctx, a.c, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: *config.Export,
		}
		return controllerutil.SetOwnerReference(item, secret, kubernetes.LandscaperScheme)
	})
	if err != nil {
		return err
	}

	item.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	return a.c.Status().Update(ctx, item)
}

func (a *actuator) getConfig(ctx context.Context, item *lsv1alpha1.DeployItem) (*mockv1alpha1.ProviderConfiguration, error) {
	config := &mockv1alpha1.ProviderConfiguration{}
	if _, _, err := serializer.NewCodecFactory(Mockscheme).UniversalDecoder().Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		a.log.Error(err, "unable to unmarshal config")
		item.Status.Conditions = lsv1alpha1helper.CreateOrUpdateConditions(item.Status.Conditions, lsv1alpha1.DeployItemValidationCondition, lsv1alpha1.ConditionFalse,
			"FailedUnmarshal", err.Error())
		_ = a.c.Status().Update(ctx, item)
		return nil, err
	}
	item.Status.Conditions = lsv1alpha1helper.CreateOrUpdateConditions(item.Status.Conditions, lsv1alpha1.DeployItemValidationCondition, lsv1alpha1.ConditionTrue,
		"SuccessfullValidation", "Successfully validated configuration")
	_ = a.c.Status().Update(ctx, item)
	return config, nil
}
