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

package components

import (
	"context"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

func NewActuator() (reconcile.Reconciler, error) {
	return &actuator{}, nil
}

type actuator struct {
	log logr.Logger
	c   client.Client
}

var _ inject.Client = &actuator{}

var _ inject.Logger = &actuator{}

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

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.log.Info("reconcile", "resource", req.NamespacedName)

	component := &v1alpha1.Component{}
	if err := a.c.Get(ctx, req.NamespacedName, component); err != nil {
		a.log.Error(err, "unable to get resource")
		return reconcile.Result{}, err
	}

	// if the component has the reconcile annotation or if the component is waiting for dependencies
	// we need to check if all required imports are satisfied
	if !v1alpha1.HasOperation(component.ObjectMeta, v1alpha1.ReoncileOperation) && component.Status.Phase != v1alpha1.ComponentPhaseWaitingDeps {
		return reconcile.Result{}, nil
	}

	ok, err := a.importsAreSatisfied(ctx, component)
	if err != nil {
		a.log.Error(err, "unable to check imports")
		return reconcile.Result{}, err
	}
	if !ok {
		a.log.V(2).Info("imports not satisfied")
		return reconcile.Result{}, nil
	}

	// as all imports are satisfied we can collect and merge all imports
	// and then start the executions

	imports, err := a.collectImports(ctx, component)
	if err != nil {
		a.log.Error(err, "unable to collect imports")
		return reconcile.Result{}, err
	}

	if err := a.runExecutions(ctx, component, imports); err != nil {
		a.log.Error(err, "error during execution")
		return reconcile.Result{}, err
	}

	// when all executions are finished and the exports are uploaded
	// we have to validate the uploaded exports
	if err := a.validateExports(ctx, component); err != nil {
		a.log.Error(err, "error during export validation")
		return reconcile.Result{}, err
	}

	// as all exports are validated, lets trigger dependant components
	if err := a.triggerDependants(ctx, component); err != nil {
		a.log.Error(err, "error during dependant trigger")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) runExecutions(ctx context.Context, component *v1alpha1.Component, imports map[string]interface{}) error {
	return nil
}

func (a *actuator) triggerDependants(ctx context.Context, component *v1alpha1.Component) error {
	return nil
}

func (a *actuator) validateExports(ctx context.Context, component *v1alpha1.Component) error {
	return nil
}

// importsAreSatisfied traverses through all components and validates if all imports are
// satisfied with the correct version
func (a *actuator) importsAreSatisfied(ctx context.Context, component *v1alpha1.Component) (bool, error) {
	return true, nil
}

// getImports traverses through all components and
// collects and merges the imports
func (a *actuator) collectImports(ctx context.Context, component *v1alpha1.Component) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}
