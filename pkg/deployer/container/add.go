// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
)

// AddControllerToManager adds all necessary deployer controllers to a controller manager.
func AddControllerToManager(hostMgr manager.Manager, lsMgr manager.Manager, config containerv1alpha1.Configuration) error {
	ctrlLogger := ctrl.Log.WithName("controllers")

	directHostClient, err := client.New(hostMgr.GetConfig(), client.Options{
		Scheme: hostMgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct client for the host cluster: %w", err)
	}
	deployer, err := NewDeployer(
		ctrlLogger.WithName("ContainerDeployer"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		directHostClient,
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
		deployer)

	err = deployerlib.Add(ctrl.Log, lsMgr, hostMgr, deployerlib.DeployerArgs{
		Type:            Type,
		Deployer:        deployer,
		TargetSelectors: config.TargetSelector,
	})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.WithPredicates(noopPredicate{})).
		Watches(src, &PodEventHandler{}).
		Complete(podRec)
}
