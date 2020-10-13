// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

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
