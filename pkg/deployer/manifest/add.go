// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(lsMgr, hostMgr manager.Manager, config manifestv1alpha2.Configuration) error {
	log := ctrl.Log.WithName("controllers").WithName("ManifestDeployer")
	d, err := NewDeployer(
		log,
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		config,
	)
	if err != nil {
		return err
	}

	return deployerlib.Add(log, lsMgr, hostMgr, deployerlib.DeployerArgs{
		Type:            Type,
		Deployer:        d,
		TargetSelectors: config.TargetSelector,
	})
}
