// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the execution controller to the controller manager
func AddControllerToManager(mgr manager.Manager) error {
	a, err := NewController(
		ctrl.Log.WithName("controllers").WithName("Executions"),
		mgr.GetClient(),
		mgr.GetScheme(),
	)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.Execution{}).
		Owns(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
