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
func AddControllerToManager(logger logging.Logger, lsMgr, hostMgr manager.Manager,
	config *config.LandscaperConfiguration, callerName string) error {

	log := logger.Reconciles("installation", "Installation")

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.Controllers.Installations.CommonControllerConfig.Workers)

	a, err := NewController(
		hostMgr.GetClient(),
		log,
		lsMgr.GetClient(),
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor("Landscaper"),
		config,
		config.Controllers.Installations.CommonControllerConfig.Workers,
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
