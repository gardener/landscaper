// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func AddControllerToManager(mgr manager.Manager, rawDeployItemPickupTimeout, rawDeployItemAbortingTimeout, rawDeployItemDefaultTimeout string) error {
	a, err := NewController(ctrl.Log.WithName("controllers").WithName("DeployItem"), mgr.GetClient(), mgr.GetScheme(), rawDeployItemPickupTimeout, rawDeployItemAbortingTimeout, rawDeployItemDefaultTimeout)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
