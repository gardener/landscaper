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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func NewActuator(registry blueprintsregistry.Registry) (reconcile.Reconciler, error) {
	return &actuator{
		registry: registry,
	}, nil
}

type actuator struct {
	log      logr.Logger
	c        client.Client
	scheme   *runtime.Scheme
	registry blueprintsregistry.Registry
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

	// remove the reconcile annotation if it exists
	if lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(exec.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.c.Update(ctx, exec); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		err := a.Ensure(ctx, exec)
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
	}

	err := a.Ensure(ctx, exec)
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

func (a *actuator) Ensure(ctx context.Context, exec *lsv1alpha1.Execution) error {
	op := execution.NewOperation(operation.NewOperation(a.log, a.c, a.scheme, a.registry, nil), exec)

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
