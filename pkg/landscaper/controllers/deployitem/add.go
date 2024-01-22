// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"
)

func AddControllerToManager(lsUncachedClient, lsCachedClient client.Client,
	logger logging.Logger,
	lsMgr manager.Manager,
	config config.DeployItemsController,
	deployItemPickupTimeout *lscore.Duration) error {

	log := logger.Reconciles("", "DeployItem")

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()),
		"numberOfWorkerThreads", config.CommonControllerConfig.Workers)

	a, err := NewController(
		lsUncachedClient, lsCachedClient,
		log,
		lsMgr.GetScheme(),
		deployItemPickupTimeout,
		config.CommonControllerConfig.Workers,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.OnlyMetadata).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(a)
}
