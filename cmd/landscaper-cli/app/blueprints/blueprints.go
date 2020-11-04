// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"

	"github.com/spf13/cobra"
)

// NewBlueprintsCommand creates a new blueprints command.
func NewBlueprintsCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "blueprints",
		Aliases: []string{"blue", "blueprint", "bp"},
		Short:   "command to interact with blueprints stored in an oci registry",
	}

	cmd.AddCommand(NewPushCommand(ctx))
	cmd.AddCommand(NewGetCommand(ctx))
	cmd.AddCommand(NewValidationCommand(ctx))
	cmd.AddCommand(NewRenderCommand(ctx))

	return cmd
}
