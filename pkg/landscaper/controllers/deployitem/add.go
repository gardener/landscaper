// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func AddControllerToManager(mgr manager.Manager, rawDeployItemPickupTimeout string) error {
	if rawDeployItemPickupTimeout == "none" {
		// currently the deploy item reconcile loop is only used for pickup timeout detection
		// so if that is disabled there is no need to watch deploy items
		return nil
	}
	a, err := NewController(ctrl.Log.WithName("controllers").WithName("DeployItem"), mgr.GetClient(), mgr.GetScheme(), rawDeployItemPickupTimeout)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployItem{}).
		Complete(a)
}
