// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/targetselector"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func NewActuator(log logr.Logger, config *containerv1alpha1.Configuration) (reconcile.Reconciler, error) {
	return &actuator{
		log:    log,
		config: config,
	}, nil
}

type actuator struct {
	log        logr.Logger
	lsClient   client.Client
	hostClient client.Client
	scheme     *runtime.Scheme
	config     *containerv1alpha1.Configuration
}

var _ inject.Client = &actuator{}

var _ inject.Scheme = &actuator{}

// InjectClients injects the current kubernetes registry into the actuator
func (a *actuator) InjectClient(c client.Client) error {
	a.lsClient = c
	return nil
}

// InjectHostClient injects the current host kubernetes registry into the actuator
func (a *actuator) InjectHostClient(c client.Client) error {
	a.hostClient = c
	return nil
}

// InjectScheme injects the current scheme into the actuator
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.log.Info("reconcile", "resource", req.NamespacedName)

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.lsClient.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			a.log.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if deployItem.Spec.Type != Type {
		return reconcile.Result{}, nil
	}

	if deployItem.Spec.Target != nil {
		target := &lsv1alpha1.Target{}
		if err := a.lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to get target for deploy item: %w", err)
		}
		if len(a.config.TargetSelector) != 0 {
			matched, err := targetselector.Match(target, a.config.TargetSelector)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("unable to match target selector: %w", err)
			}
			if !matched {
				a.log.V(5).Info("the deploy item's target has not matched the given target selector",
					"deployItem", deployItem.Name, "target", target.Name)
				return reconcile.Result{}, nil
			}
		}
	}

	if err := a.reconcile(ctx, deployItem); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem) error {
	old := deployItem.DeepCopy()
	// set failed state if the last error lasts for more than 5 minutes
	defer func() {
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(lsv1alpha1helper.GetPhaseForLastError(
			lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
			deployItem.Status.LastError,
			5*time.Minute))
	}()

	containerOp, err := New(a.log, a.lsClient, a.hostClient, a.config, deployItem)
	if err != nil {
		return err
	}

	if !kubernetes.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := a.lsClient.Update(ctx, deployItem); err != nil {
			return err
		}
		return nil
	}

	if !deployItem.DeletionTimestamp.IsZero() {
		return containerOp.Delete(ctx)
	}

	err = containerOp.Reconcile(ctx, container.OperationReconcile)
	if !reflect.DeepEqual(old.Status, deployItem.Status) {
		if err := a.lsClient.Status().Update(ctx, deployItem); err != nil {
			a.log.Error(err, "unable to update status")
		}
	}
	return err
}
