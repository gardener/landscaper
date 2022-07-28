// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/utils"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func AddControllerToManager(logger logr.Logger,
	mgr manager.Manager,
	config config.DeployItemsController,
	deployItemPickupTimeout,
	deployItemAbortingTimeout,
	deployItemDefaultTimeout *lscore.Duration) error {
	log := logger.WithName("DeployItem")
	a, err := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		deployItemPickupTimeout,
		deployItemAbortingTimeout,
		deployItemDefaultTimeout,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployItem{}).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log }).
		Complete(a)
}
