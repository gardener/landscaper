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
	"io/ioutil"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	landscaperv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
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

	inst := &v1alpha1.ComponentInstallation{}
	if err := a.c.Get(ctx, req.NamespacedName, inst); err != nil {
		a.log.Error(err, "unable to get inst installation")
		return reconcile.Result{}, err
	}

	// todo: get definition from registry
	definition := &v1alpha1.ComponentDefinition{}

	// check that all referenced definitions have a corresponding installation
	if err := a.EnsureSubInstallations(ctx, inst, definition); err != nil {
		a.log.Error(err, "unable to ensure sub installations")
		return reconcile.Result{}, err
	}

	// if the inst has the reconcile annotation or if the inst is waiting for dependencies
	// we need to check if all required imports are satisfied
	//if !v1alpha1.HasOperation(inst.ObjectMeta, v1alpha1.ReoncileOperation) && inst.Status.Phase != v1alpha1.ComponentPhaseWaitingDeps {
	//	return reconcile.Result{}, nil
	//}

	// for debugging read landscape from tmp file
	landscapeConfig := make(map[string]interface{})
	data, err := ioutil.ReadFile("./tmp/ls-config.yaml")
	if err != nil {
		return reconcile.Result{}, err
	}
	if err := yaml.Unmarshal(data, &landscapeConfig); err != nil {
		return reconcile.Result{}, err
	}

	if err := a.importsAreSatisfied(ctx, landscapeConfig, inst); err != nil {
		a.log.Error(err, "imports not satisfied")
		return reconcile.Result{}, err
	}

	// as all imports are satisfied we can collect and merge all imports
	// and then start the executions

	imports, err := a.collectImports(ctx, landscapeConfig, inst)
	if err != nil {
		a.log.Error(err, "unable to collect imports")
		return reconcile.Result{}, err
	}

	if err := a.runExecutions(ctx, inst, imports); err != nil {
		a.log.Error(err, "error during execution")
		return reconcile.Result{}, err
	}

	// when all executions are finished and the exports are uploaded
	// we have to validate the uploaded exports
	if err := a.validateExports(ctx, inst); err != nil {
		a.log.Error(err, "error during export validation")
		return reconcile.Result{}, err
	}

	// as all exports are validated, lets trigger dependant components
	if err := a.triggerDependants(ctx, inst); err != nil {
		a.log.Error(err, "error during dependant trigger")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) updateInstallationStatus(ctx context.Context, inst *v1alpha1.ComponentInstallation, updatedConditions ...v1alpha1.Condition) error {
	inst.Status.Conditions = landscaperv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	inst.Status.ObservedGeneration = inst.Generation
	if err := a.c.Status().Update(ctx, inst); err != nil {
		a.log.Error(err, "unable to update installation status")
		return err
	}
	return nil
}

func (a *actuator) triggerDependants(ctx context.Context, component *v1alpha1.ComponentInstallation) error {
	return nil
}

func (a *actuator) validateExports(ctx context.Context, component *v1alpha1.ComponentInstallation) error {
	return nil
}
