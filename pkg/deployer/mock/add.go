// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

// AddDeployerToManager adds a new helm deployers to a controller manager.
func AddDeployerToManager(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	finishedObjectCache *utils.FinishedObjectCache,
	logger logging.Logger, lsMgr, hostMgr manager.Manager, config mockv1alpha1.Configuration,
	callerName string) error {
	log := logger.WithName("mock")

	log.Info(fmt.Sprintf("Running on pod %s in namespace %s", utils.GetCurrentPodName(), utils.GetCurrentPodNamespace()))

	d, err := NewDeployer(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		log,
		config,
	)
	if err != nil {
		return err
	}

	return deployerlib.Add(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		finishedObjectCache,
		log, lsMgr, hostMgr, deployerlib.DeployerArgs{
			Name:            Name,
			Version:         version.Get().String(),
			Identity:        config.Identity,
			Type:            Type,
			Deployer:        d,
			TargetSelectors: config.TargetSelector,
		}, 5, false, callerName)
}

// NewController creates a new simple controller.
// This method should only be used for testing.
func NewController(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	finishedObjectCache *utils.FinishedObjectCache,
	log logging.Logger, scheme *runtime.Scheme, eventRecorder record.EventRecorder,
	config mockv1alpha1.Configuration, callerName string) (reconcile.Reconciler, error) {
	d, err := NewDeployer(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		log,
		config,
	)
	if err != nil {
		return nil, err
	}

	return deployerlib.NewController(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		finishedObjectCache,
		scheme, eventRecorder, scheme,
		deployerlib.DeployerArgs{
			Type:            Type,
			Deployer:        d,
			TargetSelectors: config.TargetSelector,
		}, 5, false, callerName), nil
}
