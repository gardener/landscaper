// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the execution controller to the controller manager
func AddControllerToManager(logger logr.Logger, mgr manager.Manager, config config.ExecutionsController) error {
	log := logger.WithName("Executions")
	a, err := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("Landscaper"),
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.Execution{}).
		Owns(&lsv1alpha1.DeployItem{}).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log }).
		Complete(a)
}
