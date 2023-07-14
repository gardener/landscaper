// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager adds the execution controller to the controller manager
func AddControllerToManager(logger logging.Logger, lsMgr, hostMgr manager.Manager, config config.ExecutionsController) error {
	log := logger.Reconciles("execution", "Execution")

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.CommonControllerConfig.Workers)

	a, err := NewController(
		log,
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor("Landscaper"),
		config.CommonControllerConfig.Workers,
		"executions",
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.Execution{}, builder.OnlyMetadata).
		Owns(&lsv1alpha1.DeployItem{}, builder.OnlyMetadata).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(a)
}
