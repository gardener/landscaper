// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the component overwrites controller to the controller manager.
// It is responsible for detecting timeouts in deploy items.
func AddControllerToManager(logger logr.Logger, mgr manager.Manager, cmgr *componentoverwrites.Manager) error {
	log := logger.WithName("ComponentOverwrites")
	c := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		cmgr)
	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.ComponentOverwrites{}).
		WithLogger(log).
		Complete(c)
}
