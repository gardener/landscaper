// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/gardener/landscaper/pkg/deployer/container/wait"
	"github.com/gardener/landscaper/pkg/version"
)

func NewContainerDeployerWaitCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use: "container-deployer-wait",
		Short: `Wait executor should run as a sidecar to the main container of a pod deployed by a Container Deployer.
It waits until the main container has finished, then back-ups the optional state and finally uploads the export to the DeployItem.		
`,
		Version: version.Get().GitVersion,
		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) {
	o.log.Info("Starting wait executor for container deployer", lc.KeyVersion, version.Get().GitVersion)
	if err := wait.Run(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
