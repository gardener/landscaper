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

package definitions

import (
	"context"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
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

// InjectScheme injects the current runtime scheme into the
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.log.Info("reconcile", "resource", req.NamespacedName)

	definition := &v1alpha1.ComponentDefinition{}
	if err := a.c.Get(ctx, req.NamespacedName, definition); err != nil {
		a.log.Error(err, "unable to get resource")
		return reconcile.Result{}, err
	}

	if err := a.ensureCustomTypes(ctx, definition); err != nil {
		// todo add condition
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) ensureCustomTypes(ctx context.Context, definition *v1alpha1.ComponentDefinition) error {
	if len(definition.Spec.CustomTypes) == 0 {
		a.log.V(5).Info("no custom types to create")
		return nil
	}

	for _, cType := range definition.Spec.CustomTypes {
		datatype := &v1alpha1.Type{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cType.Name,
				Namespace: definition.Namespace,
			},
			Spec: v1alpha1.TypeSpec{
				OpenAPIV3Schema: cType.OpenAPIV3Schema,
			},
		}

		if err := controllerutil.SetOwnerReference(&datatype.ObjectMeta, definition, a.scheme); err != nil {
			return err
		}

		if _, err := controllerutil.CreateOrUpdate(ctx, a.c, datatype, func() error { return nil }); err != nil {
			return err
		}

		createOrUpdateDefinitionTypeCondition(definition.Status.TypeConditions, v1alpha1.DefinitionTypeCondition{
			TypeName: cType.Name,
			TypeCondition: v1alpha1.TypeCondition{
				Type:               v1alpha1.TypeEstablished,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
			},
		})
	}

	if err := a.c.Status().Update(ctx, definition); err != nil {
		return err
	}

	return nil
}

func createOrUpdateDefinitionTypeCondition(conditions []v1alpha1.DefinitionTypeCondition, cond v1alpha1.DefinitionTypeCondition) {
	for i, foundCondition := range conditions {
		if foundCondition.TypeName == cond.TypeName {
			continue
		}

		conditions[i] = cond
		return
	}

	conditions = append(conditions, cond)
}
