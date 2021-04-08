// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
)

// AddControllerToManager adds a new mock deployer to a controller manager.
func AddControllerToManager(mgr manager.Manager, config *mockv1alpha1.Configuration) error {
	a, err := NewController(
		ctrl.Log.WithName("controllers").WithName("mock"),
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
	)
	if err != nil {
		return err
	}

	if _, err := inject.LoggerInto(ctrl.Log.WithName("controllers").WithName("MockDeployer"), a); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
