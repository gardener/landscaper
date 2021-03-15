// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager register the installation controller in a manager.
func AddControllerToManager(mgr manager.Manager, config *config.LandscaperConfiguration) error {
	a, err := NewController(
		ctrl.Log.WithName("controllers").WithName("Installations"),
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
	)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Installation{}).
		Owns(&v1alpha1.Execution{}).
		Owns(&v1alpha1.Installation{}).
		Complete(a)
}
