// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

type actuator struct {
	log        logr.Logger
	client     client.Client
	restConfig *rest.Config
	scheme     *runtime.Scheme
	config     *terraformv1alpha1.Configuration
}

var _ inject.Client = &actuator{}

var _ inject.Config = &actuator{}

var _ inject.Scheme = &actuator{}

// InjectClients injects the current kubernetes client into the actuator.
func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

// InjectConfig injects the current scheme into the actuator.
func (a *actuator) InjectConfig(config *rest.Config) error {
	a.restConfig = config
	return nil
}

// InjectScheme injects the current scheme into the actuator.
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

// Reconcile handles the reconcile flow for a terraform deploy item.
func (a *actuator) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	a.log.Info("reconcile", "resource", req.NamespacedName)

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
	// TODO: implement target types
	// if deployItem.Spec.Target != nil {
	// 	target = &lsv1alpha1.Target{}
	// 	if err := a.client.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
	// 		return reconcile.Result{}, fmt.Errorf("unable to get target for deploy item: %w", err)
	// 	}
	// 	if len(a.config.TargetSelector) != 0 {
	// 		matched, err := targetselector.Match(target, a.config.TargetSelector)
	// 		if err != nil {
	// 			return reconcile.Result{}, fmt.Errorf("unable to match target selector: %w", err)
	// 		}
	// 		if !matched {
	// 			a.log.V(5).Info("the deploy item's target has not matched the given target selector",
	// 				"deployItem", deployItem.Name, "target", target.Name)
	// 			return reconcile.Result{}, nil
	// 		}
	// 	}
	// }

	if deployItem.Status.ObservedGeneration == deployItem.Generation && !lsv1alpha1helper.HasOperation(deployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) && deployItem.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	old := deployItem.DeepCopy()
	reconcileResult, err := a.reconcile(ctx, deployItem, target)
	if !reflect.DeepEqual(old.Status, deployItem.Status) {
		if err := a.client.Status().Update(ctx, deployItem); err != nil {
			a.log.Error(err, "unable to update status")
		}
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(reconcileResult, reconcile.Result{}) {
		return reconcileResult, nil
	}

	if lsv1alpha1helper.HasOperation(deployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(deployItem.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.client.Update(ctx, deployItem); err != nil {
			deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
				"Reconcile", "RemoveReconcileAnnotation", err.Error())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (a *actuator) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (reconcile.Result, error) {
	// set failed state if the last error lasts for more than 5 minutes.
	defer func() {
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(
			lsv1alpha1helper.GetPhaseForLastError(
				lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
				deployItem.Status.LastError,
				5*time.Minute,
			))
	}()

	tf, err := New(a.log, a.client, a.restConfig, a.config, deployItem, target)
	if err != nil {
		deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
			"InitTerraformOperation", "", err.Error())
		return reconcile.Result{}, err
	}

	if !kubernetes.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := a.client.Update(ctx, deployItem); err != nil {
			deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
				"AddFinalizer", "", err.Error())
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if !deployItem.DeletionTimestamp.IsZero() {
		return tf.Reconcile(ctx, OperationDelete)
	}

	return tf.Reconcile(ctx, OperationReconcile)
}
