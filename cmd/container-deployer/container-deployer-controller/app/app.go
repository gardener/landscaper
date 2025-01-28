// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/version"
)

func NewContainerDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:     "container-deployer",
		Short:   fmt.Sprintf("Container Deployer is a controller that executes containers based on DeployItems of type %s", containerctlr.Type),
		Version: version.Get().GitVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Complete(); err != nil {
				return err
			}
			return options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())
	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.DeployerOptions.Log.Info("Starting Container Deployer", lc.KeyVersion, version.Get().GitVersion)

	callerName := "container"
	controllerName := "deployitem"

	gc, err := containerctlr.AddControllerToManager(
		o.DeployerOptions.LsUncachedClient, o.DeployerOptions.LsCachedClient, o.DeployerOptions.HostUncachedClient, o.DeployerOptions.HostCachedClient,
		o.DeployerOptions.FinishedObjectCache,
		o.DeployerOptions.Log,
		o.DeployerOptions.HostMgr,
		o.DeployerOptions.LsMgr,
		o.Config,
		callerName, controllerName)
	if err != nil {
		return fmt.Errorf("unable to setup container controller")
	}

	o.DeployerOptions.Log.Info("Starting container deployer manager")

	if gc == nil {
		return o.DeployerOptions.StartManagers(ctx)
	} else {
		return o.DeployerOptions.StartManagers(ctx, gc)
	}
}
