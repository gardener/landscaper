// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

// AddControllerToManager adds all necessary deployer controllers to a controller manager.
func AddControllerToManager(logger logging.Logger, hostMgr, lsMgr manager.Manager, config containerv1alpha1.Configuration,
	callerName string) (*GarbageCollector, error) {
	log := logger.WithName("container")

	lockingEnabled := config.HPAConfiguration != nil && config.HPAConfiguration.MaxReplicas > 1

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controller.Workers,
		"lockingEnabled", lockingEnabled)

	directHostClient, err := client.New(hostMgr.GetConfig(), client.Options{
		Scheme: hostMgr.GetScheme(),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create direct client for the host cluster: %w", err)
	}
	containerDeployer, err := NewDeployer(
		log,
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		directHostClient,
		config)
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	if config.GarbageCollection.Disable {
		log.Info("GarbageCollector disabled")
		return nil, nil
	}

	keepPods := config.DebugOptions != nil && config.DebugOptions.KeepPod
	gc := NewGarbageCollector(log.WithName("garbageCollector"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		config.Identity,
		config.Namespace,
		config.GarbageCollection,
		keepPods)

	return gc, nil
}
