// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
)

// AddActuatorToManager adds the terraform deployer actuator to the manager.
func AddActuatorToManager(mgr manager.Manager, config *terraformv1alpha1.Configuration) error {
	a := &actuator{
		log:    ctrl.Log.WithName("controllers").WithName("TerraformDeployer"),
		config: config,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
