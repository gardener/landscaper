// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/version"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
)

// AddControllerToManager adds all necessary deployer controllers to a controller manager.
func AddControllerToManager(logger logr.Logger, hostMgr, lsMgr manager.Manager, config containerv1alpha1.Configuration) error {
	ctrlLogger := logger.WithName("ContainerDeployer")

	directHostClient, err := client.New(hostMgr.GetConfig(), client.Options{
		Scheme: hostMgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct client for the host cluster: %w", err)
	}
	deployer, err := NewDeployer(
		ctrlLogger,
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
		lsMgr.GetEventRecorderFor("Landscaper"),
		config,
		deployer)

	options := controller.Options{
		MaxConcurrentReconciles: config.Controller.Workers,
	}
	if config.Controller.CacheSyncTimeout != nil {
		options.CacheSyncTimeout = config.Controller.CacheSyncTimeout.Duration
	}

	err = deployerlib.Add(ctrl.Log, lsMgr, hostMgr, deployerlib.DeployerArgs{
		Name:            Name,
		Version:         version.Get().String(),
		Identity:        config.Identity,
		Type:            Type,
		Deployer:        deployer,
		TargetSelectors: config.TargetSelector,
		Options:         options,
	})
	if err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.WithPredicates(noopPredicate{})).
		Watches(src, &PodEventHandler{}).
		Complete(podRec); err != nil {
		return err
	}

	if config.GarbageCollection.Disable {
		logger.Info("GarbageCollector disabled")
		return nil
	}
	return NewGarbageCollector(logger.WithName("GarbageCollector"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		config.Identity,
		config.Namespace,
		config.GarbageCollection).
		Add(hostMgr, config.DebugOptions != nil && config.DebugOptions.KeepPod)
}
