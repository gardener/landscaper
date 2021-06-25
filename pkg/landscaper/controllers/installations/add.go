// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllerToManager register the installation Controller in a manager.
func AddControllerToManager(logger logr.Logger, mgr manager.Manager, overwriter componentoverwrites.Overwriter, config *config.LandscaperConfiguration) error {
	log := logger.WithName("Installations")
	a, err := NewController(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("Landscaper"),
		overwriter,
		config,
	)
	if err != nil {
		return err
	}

	return builder.ControllerManagedBy(mgr).
		For(&v1alpha1.Installation{}).
		Owns(&v1alpha1.Execution{}).
		Owns(&v1alpha1.Installation{}).
		WithLogger(log).
		Complete(a)
}
