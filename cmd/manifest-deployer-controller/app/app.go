// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
)

func NewManifestDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:          "manifest-deployer",
		Short:        fmt.Sprintf("Manifest Deployer is a controller that applies kubernetes manifests based on DeployItems of type %s", manifestctlr.Type),
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
	if err := manifestctlr.AddDeployerToManager(o.DeployerOptions.Log, o.DeployerOptions.LsMgr, o.DeployerOptions.HostMgr, o.Config); err != nil {
		return fmt.Errorf("unable to setup helm controller")
	}
	return o.DeployerOptions.StartManagers(ctx)
}
