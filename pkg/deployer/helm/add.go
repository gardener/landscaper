// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(lsMgr, hostMgr manager.Manager, config helmv1alpha1.Configuration) error {
	log := ctrl.Log.WithName("controllers").WithName("HelmDeployer")
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
