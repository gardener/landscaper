// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the component overwrites controller to the controller manager.
// It is responsible for detecting timeouts in deploy items.
func AddControllerToManager(logger logging.Logger, mgr manager.Manager, cmgr *componentoverwrites.Manager, config config.ComponentOverwritesController) error {
	log := logger.WithName("ComponentOverwrites").WithValues(lc.KeyReconciledResourceKind, "ComponentOverwrite")
	c := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		cmgr)
	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.ComponentOverwrites{}).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(c)
}
