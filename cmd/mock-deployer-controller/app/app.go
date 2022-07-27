// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	mockctrl "github.com/gardener/landscaper/pkg/deployer/mock"
)

func NewMockDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:          "mock-deployer",
		Short:        fmt.Sprintf("Mock Deployer is a controller that mocks the behavior of deploy items of type %s", mockctrl.Type),
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
	if err := mockctrl.AddDeployerToManager(o.DeployerOptions.Log, o.DeployerOptions.LsMgr, o.DeployerOptions.HostMgr, o.Config); err != nil {
		return fmt.Errorf("unable to setup mock controller")
	}
	return o.DeployerOptions.StartManagers(ctx)
}
