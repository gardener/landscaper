// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/config"
)

// AddControllerToManager adds the context defaulterController to the defaulterController manager.
// That defaulterController watches namespaces and creates the default context object in every namespace.
func AddControllerToManager(lsUncachedClient, lsCachedClient client.Client,
	logger logging.Logger, lsMgr manager.Manager, config *config.LandscaperConfiguration) error {
	log := logger.Reconciles("context", "Namespace")
	if config.Controllers.Contexts.Config.Default.Disable {
		log.Info("Default Context controller is disabled")
		return nil
	}

	a, err := NewDefaulterController(
		lsUncachedClient, lsCachedClient,
		log,
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor("Landscaper"),
		config.Controllers.Contexts.Config,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(lsMgr).
		For(&corev1.Namespace{}, builder.OnlyMetadata).
		WithOptions(utils.ConvertCommonControllerConfigToControllerOptions(config.Controllers.Contexts.CommonControllerConfig)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(a)
}
