// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddToManager adds the agent to the provided manager.
func AddToManager(ctx context.Context, logger logr.Logger, lsMgr manager.Manager, hostMgr manager.Manager, config config.AgentConfiguration) error {
	log := logger.WithName("agent")
	// create direct client for the agent to ensure the landscaper resources
	lsClient, err := client.New(lsMgr.GetConfig(), client.Options{
		Scheme: lsMgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct landscaper kubernetes client: %w", err)
	}
	hostClient, err := client.New(hostMgr.GetConfig(), client.Options{
		Scheme: hostMgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct landscaper kubernetes client: %w", err)
	}
	agent := New(log,
		lsMgr.GetClient(),
		lsMgr.GetConfig(),
		lsMgr.GetScheme(),
		hostMgr.GetClient(),
		hostMgr.GetConfig(),
		hostMgr.GetScheme(),
		config,
	)

	if _, err := agent.EnsureLandscaperResources(ctx, lsClient, hostClient); err != nil {
		return fmt.Errorf("unable to ensure landscaper resources: %w", err)
	}
	if _, err := agent.EnsureHostResources(ctx, hostClient); err != nil {
		return fmt.Errorf("unable to ensure host resources: %w", err)
	}

	err = builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.Environment{}).
		WithLogger(log).
		Complete(agent)
	if err != nil {
		return err
	}

	// register helm deployer.
	helmConfig := helmv1alpha1.Configuration{}
	helmConfig.Identity = fmt.Sprintf("agent-helm-deployer-%s", config.Name)
	helmConfig.OCI = config.OCI
	helmConfig.TargetSelector = []lsv1alpha1.TargetSelector{
		{
			Targets: []lsv1alpha1.ObjectReference{{Name: config.Name}},
		},
	}
	if err := helmctlr.AddDeployerToManager(log, lsMgr, hostMgr, helmConfig); err != nil {
		return err
	}

	return nil
}
