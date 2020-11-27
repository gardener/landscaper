// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	terraformv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/terraform/v1alpha1"
)

func AddActuatorToManager(mgr manager.Manager, config *terraformv1alpha1.Configuration) error {
	a, err := NewActuator(ctrl.Log.WithName("controllers").WithName("TerraformDeployer"), config)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
