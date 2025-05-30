// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(
	lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	finishedObjectCache *utils.FinishedObjectCache,
	logger logging.Logger, lsMgr, hostMgr manager.Manager,
	config helmv1alpha1.Configuration, callerName, controllerName string) error {
	log := logger.WithName("helm")

	lockingEnabled := config.HPAConfiguration != nil && config.HPAConfiguration.MaxReplicas > 1

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controller.Workers,
		"lockingEnabled", lockingEnabled)

	// check if allowed to access
	problemHandler := utils.GetCriticalProblemsHandler()
	if err := problemHandler.AccessAllowed(context.Background(), hostUncachedClient); err != nil {
		return err
	}
	log.Info("access to critical problems allowed")

	d, err := NewDeployer(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient, lsMgr.GetConfig(), log, config)
	if err != nil {
		return err
	}

	options := controller.Options{
		MaxConcurrentReconciles: config.Controller.Workers,
	}
	if config.Controller.CacheSyncTimeout != nil {
		options.CacheSyncTimeout = config.Controller.CacheSyncTimeout.Duration
	}

	return deployerlib.Add(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		finishedObjectCache,
		log, lsMgr, hostMgr, deployerlib.DeployerArgs{
			Name:            Name,
			Version:         version.Get().String(),
			Identity:        config.Identity,
			Type:            Type,
			Deployer:        d,
			TargetSelectors: config.TargetSelector,
			Options:         options,
		}, config.Controller.Workers, lockingEnabled, callerName, controllerName)
}
