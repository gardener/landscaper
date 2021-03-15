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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// NewController creates a new execution controller that reconcile Execution resources.
func NewController(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme) (reconcile.Reconciler, error) {
	return &controller{
		log:    log,
		c:      kubeClient,
		scheme: scheme,
	}, nil
}

type controller struct {
	log    logr.Logger
	c      client.Client
	scheme *runtime.Scheme
}

func (a *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
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

func (a *controller) Ensure(ctx context.Context, exec *lsv1alpha1.Execution, forceReconcile bool) error {
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
