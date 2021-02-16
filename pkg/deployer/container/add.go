// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
)

func AddActuatorToManager(hostMgr manager.Manager, lsMgr manager.Manager, config *containerv1alpha1.Configuration) error {
	ctrlLogger := ctrl.Log.WithName("controllers")
	diRec, err := NewDeployItemReconciler(
		ctrlLogger.WithName("ContainerDeployer"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		config)
	if err != nil {
		return err
	}

	src := source.NewKindWithCache(&corev1.Pod{}, hostMgr.GetCache())
	podRec := NewPodReconciler(
		ctrlLogger.WithName("PodReconciler"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		config,
		diRec)

	err = ctrl.NewControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}).
		Complete(diRec)
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.WithPredicates(noopPredicate{})).
		Watches(src, &PodEventHandler{}).
		Complete(podRec)
}
