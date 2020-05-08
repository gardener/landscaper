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
	"github.com/pkg/errors"
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

	// Run component
	if component.Status.Phase == "" || component.Status.Phase == v1alpha1.ComponentPhaseInit {
		for i, executorState := range component.Status.Executors {
			if executorState.Phase == "" || executorState.Phase == v1alpha1.ComponentPhaseInit {

				// get definition
				definition := &v1alpha1.ComponentDefinition{}
				if err := a.c.Get(ctx, client.ObjectKey{Name: component.Spec.DefinitionRef, Namespace: component.Namespace}, definition); err != nil {
					a.log.Error(err, "unable to get definition")
					return reconcile.Result{}, err
				}
				// run executor
				executor := definition.Spec.Executors[i]

				switch executor.Type {
				case v1alpha1.ExecutionTypeScript:

				default:
					return reconcile.Result{}, errors.Errorf("unknown executor %s", executor.Type)
				}

			}
		}
	}

	// If state is progressing check executor
	if component.Status.Phase == v1alpha1.ComponentPhaseProgressing {

	}

	return reconcile.Result{}, nil
}
