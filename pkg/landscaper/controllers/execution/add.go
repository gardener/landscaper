// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"
)

// AddControllerToManager adds the execution controller to the controller manager
func AddControllerToManager(logger logging.Logger, lsMgr, hostMgr manager.Manager, config *config.LandscaperConfiguration) error {
	log := logger.Reconciles("execution", "Execution")

	lockingEnabled := lock.IsLockingEnabledForMainControllers(config)

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controllers.Executions.CommonControllerConfig.Workers,
		"lockingEnabled", lockingEnabled)

	a, err := NewController(
		log,
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor("Landscaper"),
		config.Controllers.Executions.CommonControllerConfig.Workers,
		lockingEnabled,
		"executions",
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.Execution{}, builder.OnlyMetadata).
		Owns(&lsv1alpha1.DeployItem{}, builder.OnlyMetadata).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.Controllers.Executions.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(a)
}
