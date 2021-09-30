// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
)

// AddControllerToManager adds the context defaulterController to the defaulterController manager.
// That defaulterController watches namespaces and creates the default context object in every namespace.
func AddControllerToManager(logger logr.Logger, mgr manager.Manager, config *config.LandscaperConfiguration) error {
	log := logger.WithName("Context")
	if config.Controllers.Context.Config.Default.Disable {
		log.Info("Default Context controller is disabled")
		return nil
	}

	a, err := NewDefaulterController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("Landscaper"),
		config.Controllers.Context.Config,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithLogger(log).
		Complete(a)
}
