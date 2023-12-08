// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"

	"github.com/gardener/landscaper/pkg/deployer/lib/cmd"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(do *cmd.DefaultOptions,
	config helmv1alpha1.Configuration, callerName string) error {
	log := do.Log.WithName("helm")

	lockingEnabled := config.HPAConfiguration != nil && config.HPAConfiguration.MaxReplicas > 1

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controller.Workers,
		"lockingEnabled", lockingEnabled)

	lsClient := do.LsClient
	if lsClient == nil {
		lsClient = do.LsMgr.GetClient()
	}

	hostClient := do.HostClient
	if hostClient == nil {
		hostClient = do.HostMgr.GetClient()
	}

	d, err := NewDeployer(
		log,
		lsClient,
		hostClient,
		config,
	)
	if err != nil {
		return err
	}

	options := controller.Options{
		MaxConcurrentReconciles: config.Controller.Workers,
	}
	if config.Controller.CacheSyncTimeout != nil {
		options.CacheSyncTimeout = config.Controller.CacheSyncTimeout.Duration
	}

	return deployerlib.Add(log, do.LsMgr, do.HostMgr, deployerlib.DeployerArgs{
		Name:            Name,
		Version:         version.Get().String(),
		Identity:        config.Identity,
		Type:            Type,
		Deployer:        d,
		TargetSelectors: config.TargetSelector,
		Options:         options,
	}, config.Controller.Workers, lockingEnabled, callerName)
}
