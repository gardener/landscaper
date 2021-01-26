// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
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

// InjectClients injects the current kubernetes client into the actuator
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

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.log.Info("reconcile", "resource", req.NamespacedName)

	exec := &lsv1alpha1.Execution{}
	if err := a.c.Get(ctx, req.NamespacedName, exec); err != nil {
		if apierrors.IsNotFound(err) {
			a.log.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	old := exec.DeepCopy()

	isForceReconcileOperation := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
	isReconcileOperation := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ReconcileOperation)

	if isForceReconcileOperation || isReconcileOperation {
		a.log.Info("reconcile annotation found", "execution", req.String(),
			"operation", lsv1alpha1helper.GetOperation(exec.ObjectMeta))
		delete(exec.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.c.Update(ctx, exec); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	err := a.Ensure(ctx, exec, isForceReconcileOperation)
	if !reflect.DeepEqual(exec.Status, old.Status) {
		if err2 := a.c.Status().Update(ctx, exec); err2 != nil {
			if err != nil {
				err2 = errors.Wrapf(err, "update error: %s", err.Error())
			}
			return reconcile.Result{}, err2
		}
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) Ensure(ctx context.Context, exec *lsv1alpha1.Execution, forceReconcile bool) error {
	op := execution.NewOperation(operation.NewOperation(a.log, a.c, a.scheme, nil), exec,
		forceReconcile)

	if exec.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(exec, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
		return a.c.Update(ctx, exec)
	}

	if !exec.DeletionTimestamp.IsZero() {
		return op.Delete(ctx)
	}

	if err := op.Reconcile(ctx); err != nil {
		return err
	}

	return nil
}
