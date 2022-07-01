// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the component overwrites controller to the controller manager.
// It is responsible for detecting timeouts in deploy items.
func AddControllerToManager(logger logr.Logger, mgr manager.Manager, cmgr *componentoverwrites.Manager, config config.ComponentOverwritesController) error {
	log := logger.WithName("ComponentOverwrites")
	c := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		cmgr)
	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.ComponentOverwrites{}).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log }).
		Complete(c)
}
