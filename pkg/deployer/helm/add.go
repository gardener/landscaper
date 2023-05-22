// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/version"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(logger logging.Logger, lsMgr, hostMgr manager.Manager, config helmv1alpha1.Configuration) error {
	log := logger.WithName("helm")

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()))

	d, err := NewDeployer(
		log,
		lsMgr.GetClient(),
		hostMgr.GetClient(),
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

	return deployerlib.Add(log, lsMgr, hostMgr, deployerlib.DeployerArgs{
		Name:            Name,
		Version:         version.Get().String(),
		Identity:        config.Identity,
		Type:            Type,
		Deployer:        d,
		TargetSelectors: config.TargetSelector,
		Options:         options,
	})
}
