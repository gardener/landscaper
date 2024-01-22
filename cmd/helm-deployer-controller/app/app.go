// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/spf13/cobra"

	helmctrl "github.com/gardener/landscaper/pkg/deployer/helm"
	"github.com/gardener/landscaper/pkg/version"
)

func NewHelmDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:          "helm-deployer",
		Short:        fmt.Sprintf("Helm Deployer is a controller that deploys helm charts based on DeployItems of type %s", helmctrl.Type),
		SilenceUsage: true,
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
	o.DeployerOptions.Log.Info("Starting helm deployer", lc.KeyVersion, version.Get().GitVersion)
	if err := helmctrl.AddDeployerToManager(
		o.DeployerOptions.LsUncachedClient, o.DeployerOptions.LsCachedClient, o.DeployerOptions.HostUncachedClient, o.DeployerOptions.HostCachedClient,
		o.DeployerOptions.FinishedObjectCache,
		o.DeployerOptions.Log, o.DeployerOptions.LsMgr, o.DeployerOptions.HostMgr,
		o.Config, "helm"); err != nil {
		return fmt.Errorf("unable to setup helm controller")
	}

	o.DeployerOptions.Log.Info("Starting helm deployer manager")
	return o.DeployerOptions.StartManagers(ctx)
}
