// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllersToManager adds all deployer registration related deployers to the manager.
func AddControllersToManager(lsUncachedClient, lsCachedClient client.Client,
	logger logging.Logger, lsMgr manager.Manager, config *config.LandscaperConfiguration) error {
	log := logger.Reconciles("environment", "Environment")
	env := NewEnvironmentController(
		lsUncachedClient, lsCachedClient,
		log,
		lsMgr.GetScheme(),
		config,
	)

	err := builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.Environment{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(env)
	if err != nil {
		return fmt.Errorf("unable to register environment controller: %w", err)
	}

	log = logger.Reconciles("deployerRegistration", "DeployerRegistration")
	deployerReg := NewDeployerRegistrationController(
		lsUncachedClient, lsCachedClient,
		log,
		lsMgr.GetScheme(),
		config,
	)

	err = builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployerRegistration{}, builder.OnlyMetadata).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(deployerReg)
	if err != nil {
		return fmt.Errorf("unable to register deployer registration controller: %w", err)
	}

	log = logger.Reconciles("deployerRegistration", "Installation")
	inst := NewInstallationController(
		lsUncachedClient, lsCachedClient,
		log,
		lsMgr.GetScheme(),
		config,
	)

	err = builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.Installation{}, builder.OnlyMetadata).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(inst)
	if err != nil {
		return fmt.Errorf("unable to register installation controller: %w", err)
	}
	return nil
}
