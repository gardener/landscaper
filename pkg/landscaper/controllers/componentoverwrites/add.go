// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the component overwrites controller to the controller manager.
// It is responsible for detecting timeouts in deploy items.
func AddControllerToManager(mgr manager.Manager, cmgr *componentoverwrites.Manager) error {
	c := NewController(
		ctrl.Log.WithName("controllers").WithName("ComponentOverwrites"),
		mgr.GetClient(),
		mgr.GetScheme(),
		cmgr)
	return ctrl.NewControllerManagedBy(mgr).
		For(&lsv1alpha1.ComponentOverwrites{}).
		Complete(c)
}
