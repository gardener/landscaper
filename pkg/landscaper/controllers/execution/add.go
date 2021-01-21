// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func AddActuatorToManager(mgr manager.Manager) error {
	a, err := NewActuator()
	if err != nil {
		return err
	}

	if _, err := inject.LoggerInto(ctrl.Log.WithName("controllers").WithName("Execution"), a); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.Execution{}).
		Owns(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
