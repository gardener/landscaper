// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/blueprints"
	"github.com/gardener/landscaper/cmd/landscaper-cli/app/componentdescriptor"
	"github.com/gardener/landscaper/cmd/landscaper-cli/app/config"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/version"

	"github.com/spf13/cobra"
)

func NewLandscaperCliCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "landscaper-cli",
		Short: "landscaper cli",
		PreRun: func(cmd *cobra.Command, args []string) {
			log, err := logger.NewCliLogger()
			if err != nil {
				fmt.Println("unable to setup logger")
				fmt.Println(err.Error())
				os.Exit(1)
			}
			logger.SetLogger(log)
		},
	}

	logger.InitFlags(cmd.Flags())

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(config.NewConfigCommand(ctx))
	cmd.AddCommand(blueprints.NewBlueprintsCommand(ctx))
	cmd.AddCommand(componentdescriptor.NewComponentsCommand(ctx))

	return cmd
}

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "v",
		Run: func(cmd *cobra.Command, args []string) {
			v := version.Get()
			fmt.Printf("%#v", v)
		},
	}
}
