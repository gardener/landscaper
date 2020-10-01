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

package container

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/source"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
)

func AddActuatorToManager(hostMgr manager.Manager, landscaperMgr manager.Manager, config *containerv1alpha1.Configuration) error {
	a, err := NewActuator(ctrl.Log.WithName("controllers").WithName("ContainerDeployer"), config)
	if err != nil {
		return err
	}
	if err := hostMgr.Add(&hostRunnable{a: a}); err != nil {
		return err
	}

	src := source.NewKindWithCache(&corev1.Pod{}, hostMgr.GetCache())
	hdler := &handler.EnqueueRequestForOwner{
		OwnerType:    &lsv1alpha1.DeployItem{},
		IsController: true,
	}

	return ctrl.NewControllerManagedBy(landscaperMgr).
		For(&lsv1alpha1.DeployItem{}).
		Watches(src, hdler).
		Complete(a)
}

// HostClient is used by the ControllerManager to inject the host client into teh actuator
type HostClient interface {
	InjectHostClient(client.Client) error
}

// hostRunnable is a dummy runnable function that is used to inject the host lsClient into the actuator.
type hostRunnable struct {
	a reconcile.Reconciler
}

var _ manager.Runnable = &hostRunnable{}
var _ inject.Client = &hostRunnable{}

func (_ hostRunnable) Start(<-chan struct{}) error { return nil }

func (r hostRunnable) InjectClient(client client.Client) error {
	if s, ok := r.a.(HostClient); ok {
		return s.InjectHostClient(client)
	}
	return nil
}
