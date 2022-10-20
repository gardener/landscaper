// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/selection"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddToManager adds the agent to the provided manager.
func AddToManager(ctx context.Context, logger logging.Logger, lsMgr manager.Manager, hostMgr manager.Manager, config config.AgentConfiguration) error {
	log := logger.WithName("agent").WithValues("targetEnvironment", config.Name)
	ctx = logging.NewContext(ctx, log)
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
	agent := New(lsMgr.GetClient(),
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
	if _, err := agent.EnsureHostResources(ctx, hostClient, lsClient); err != nil {
		return fmt.Errorf("unable to ensure host resources: %w", err)
	}

	err = builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.Environment{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Reconciles("environment", "Environment").Logr() }).
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
			Annotations: []lsv1alpha1.Requirement{
				{
					Key:      lsv1alpha1.DeployerEnvironmentTargetAnnotationName,
					Operator: selection.Equals,
					Values:   []string{config.Name},
				},
				{
					Key:      lsv1alpha1.DeployerOnlyTargetAnnotationName,
					Operator: selection.Exists,
				},
			},
		},
	}
	if err := helmctlr.AddDeployerToManager(log, lsMgr, hostMgr, helmConfig); err != nil {
		return err
	}

	return nil
}
