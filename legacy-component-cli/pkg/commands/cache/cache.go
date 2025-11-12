// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cachecmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/spf13/cobra"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

// NewCacheCommand creates a new cache command.
func NewCacheCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "cache",
		Run: func(cmd *cobra.Command, args []string) {
			opts := &InfoOptions{}
			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}
	cmd.AddCommand(NewInfoCommand(ctx))
	cmd.AddCommand(NewPruneCommand(ctx))
	return cmd
}
