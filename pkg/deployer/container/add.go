// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

// AddControllerToManager adds all necessary deployer controllers to a controller manager.
func AddControllerToManager(logger logging.Logger, hostMgr, lsMgr manager.Manager, config containerv1alpha1.Configuration,
	callerName string) error {
	log := logger.WithName("container")

	lockingEnabled := config.HPAConfiguration != nil && config.HPAConfiguration.MaxReplicas > 1

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controller.Workers,
		"lockingEnabled", lockingEnabled)

	directHostClient, err := client.New(hostMgr.GetConfig(), client.Options{
		Scheme: hostMgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct client for the host cluster: %w", err)
	}
	containerDeployer, err := NewDeployer(
		log,
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		directHostClient,
		config)
	if err != nil {
		return err
	}

	podRec := NewPodReconciler(
		log.WithName("podReconciler"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		lsMgr.GetEventRecorderFor("Landscaper"),
		config,
		containerDeployer,
		Type,
		lockingEnabled,
		callerName+"pod",
		config.TargetSelector)

	options := controller.Options{
		MaxConcurrentReconciles: config.Controller.Workers,
	}
	if config.Controller.CacheSyncTimeout != nil {
		options.CacheSyncTimeout = config.Controller.CacheSyncTimeout.Duration
	}

	err = deployerlib.Add(log, lsMgr, hostMgr, deployerlib.DeployerArgs{
		Name:            Name,
		Version:         version.Get().String(),
		Identity:        config.Identity,
		Type:            Type,
		Deployer:        containerDeployer,
		TargetSelectors: config.TargetSelector,
		Options:         options,
	}, config.Controller.Workers, lockingEnabled, callerName)
	if err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(hostMgr).
		For(&corev1.Pod{}, builder.WithPredicates(newNamespaceAndAnnotationPredicate()), builder.OnlyMetadata).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(podRec); err != nil {
		return err
	}

	if config.GarbageCollection.Disable {
		log.Info("GarbageCollector disabled")
		return nil
	}
	return NewGarbageCollector(log.WithName("garbageCollector"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		config.Identity,
		config.Namespace,
		config.GarbageCollection).
		Add(hostMgr, config.DebugOptions != nil && config.DebugOptions.KeepPod)
}
