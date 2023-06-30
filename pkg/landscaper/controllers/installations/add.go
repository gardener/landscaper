// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager register the installation Controller in a manager.
func AddControllerToManager(logger logging.Logger, mgr manager.Manager, config *config.LandscaperConfiguration) error {
	log := logger.Reconciles("installation", "Installation")

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerTreads", config.Controllers.Installations.CommonControllerConfig.Workers)

	a, err := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("Landscaper"),
		config,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(mgr).
		For(&v1alpha1.Installation{}).
		Owns(&v1alpha1.Execution{}).
		Owns(&v1alpha1.Installation{}).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.Controllers.Installations.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(a)
}
