// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/spf13/cobra"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	initpkg "github.com/gardener/landscaper/pkg/deployer/container/init"
	"github.com/gardener/landscaper/pkg/version"
)

func NewContainerDeployerInitCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:     "container-deployer-init",
		Short:   "Init executor bootstraps a container that is deployed by a container deployer",
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
	o.log.Info("Starting init executor for container deployer", lc.KeyVersion, version.Get().GitVersion)
	if err := initpkg.Run(ctx, o.log, osfs.New()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
