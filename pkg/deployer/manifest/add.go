// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/version"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(logger logr.Logger, lsMgr, hostMgr manager.Manager, config manifestv1alpha2.Configuration) error {
	log := logger.WithName("ManifestDeployer")
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
