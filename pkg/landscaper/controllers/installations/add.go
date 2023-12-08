// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"fmt"

	"github.com/gardener/landscaper/pkg/utils/lock"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"
)

// AddControllerToManager register the installation Controller in a manager.
func AddControllerToManager(logger logging.Logger, lsMgr, hostMgr manager.Manager, config *config.LandscaperConfiguration, callerName string,
	indepLsClient, indepHostClient client.Client) error {
	log := logger.Reconciles("installation", "Installation")

	lockingEnabled := lock.IsLockingEnabledForMainControllers(config)

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controllers.Installations.CommonControllerConfig.Workers,
		"lockingEnabled", lockingEnabled)

	hostClient := indepHostClient
	if indepHostClient != nil {
		hostClient = indepHostClient
	}

	lsClient := indepLsClient
	if indepLsClient != nil {
		lsClient = indepLsClient
	}

	a, err := NewController(
		hostClient,
		log,
		lsClient,
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor("Landscaper"),
		config,
		config.Controllers.Installations.CommonControllerConfig.Workers,
		lockingEnabled,
		callerName,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(lsMgr).
		For(&v1alpha1.Installation{}, builder.OnlyMetadata).
		Owns(&v1alpha1.Execution{}, builder.OnlyMetadata).
		Owns(&v1alpha1.Installation{}, builder.OnlyMetadata).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.Controllers.Installations.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(a)
}
