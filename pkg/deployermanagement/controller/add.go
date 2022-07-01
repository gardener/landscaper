// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllersToManager adds all deployer registration related deployers to the manager.
func AddControllersToManager(log logr.Logger, mgr manager.Manager, config *config.LandscaperConfiguration) error {
	env := NewEnvironmentController(
		log.WithName("Environment"),
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
	)

	err := builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.Environment{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.WithName("Environment") }).
		Complete(env)
	if err != nil {
		return fmt.Errorf("unable to register environment controller: %w", err)
	}

	deployerReg := NewDeployerRegistrationController(
		log.WithName("DeployerRegistration"),
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
	)

	err = builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.DeployerRegistration{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.WithName("DeployerRegistration") }).
		Complete(deployerReg)
	if err != nil {
		return fmt.Errorf("unable to register deployer registration controller: %w", err)
	}

	inst := NewInstallationController(
		log.WithName("DeployerRegistration"),
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
	)

	err = builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.Installation{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.WithName("DeployerRegistration") }).
		Complete(inst)
	if err != nil {
		return fmt.Errorf("unable to register installation controller: %w", err)
	}
	return nil
}
