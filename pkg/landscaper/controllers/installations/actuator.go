// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package installations

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/utils/componentrepository"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func NewActuator(registry registry.Registry, compRepo componentrepository.Client) (reconcile.Reconciler, error) {
	op := &operation.Operation{}
	if err := op.InjectRegistry(registry); err != nil {
		return nil, err
	}
	if err := op.InjectComponentRepository(compRepo); err != nil {
		return nil, err
	}
	return &actuator{
		Interface: op,
	}, nil
}

type actuator struct {
	operation.Interface
}

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.Log().Info("reconcile", "resource", req.NamespacedName)

	inst := &lsv1alpha1.Installation{}
	if err := a.Client().Get(ctx, req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			a.Log().V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if inst.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := a.Client().Update(ctx, inst); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{}, nil
	}

	// remove the reconcile annotation if it exists
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.Client().Update(ctx, inst); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		if err := a.reconcile(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		if err := a.reconcile(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: handle abort..
	}

	if lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) && inst.Status.ObservedGeneration == inst.Generation {
		return reconcile.Result{}, nil
	}

	if err := a.reconcile(ctx, inst); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (a *actuator) reconcile(ctx context.Context, inst *lsv1alpha1.Installation) error {
	old := inst.DeepCopy()

	intBlueprint, err := blueprints.Resolve(ctx, a.Interface, inst.Spec.BlueprintRef)
	if err != nil {
		return err
	}

	internalInstallation, err := installations.New(inst, intBlueprint)
	if err != nil {
		return errors.Wrap(err, "unable to create internal representation of installation")
	}

	datatypeList := &lsv1alpha1.DataTypeList{}
	if err := a.Client().List(ctx, datatypeList); err != nil {
		return errors.Wrap(err, "unable to list all datatypes")
	}
	datatypes, err := datatype.CreateDatatypesMap(datatypeList.Items)
	if err != nil {
		return errors.Wrap(err, "unable to parse datatypes")
	}

	instOp, err := installations.NewInstallationOperationFromOperation(ctx, a.Interface, datatypes, internalInstallation)
	if err != nil {
		return errors.Wrap(err, "unable to create installation operation")
	}

	if !inst.DeletionTimestamp.IsZero() {
		err := EnsureDeletion(ctx, instOp)
		if err != nil && !reflect.DeepEqual(inst.Status, old.Status) {
			if err2 := a.Client().Status().Update(ctx, inst); err2 != nil {
				return errors.Wrapf(err, "update error: %s", err2.Error())
			}
		}
		return err
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		inst.Status.Phase = lsv1alpha1.ComponentPhasePending
		if err := a.StartNewReconcile(ctx, instOp, internalInstallation); err != nil {
			return err
		}

		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.Client().Update(ctx, inst); err != nil {
			return err
		}

		inst.Status.ObservedGeneration = inst.Generation
		inst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

		// need to return and not continue with export validation
		return err
	}

	err = a.Ensure(ctx, instOp, internalInstallation)
	if !reflect.DeepEqual(inst.Status, old.Status) {
		if err2 := a.Client().Status().Update(ctx, inst); err2 != nil {
			if err != nil {
				err2 = errors.Wrapf(err, "update error: %s", err.Error())
			}
			return err2
		}
	}
	return err
}
