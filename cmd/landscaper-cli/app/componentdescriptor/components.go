// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentdescriptor

import (
	"context"

	"github.com/spf13/cobra"
)

// NewComponentsCommand creates a new components command.
func NewComponentsCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "componentdescriptor",
		Aliases: []string{"cd", "comp"},
		Short:   "command to interact with component descriptors stored in an oci registry",
	}

	cmd.AddCommand(NewPushCommand(ctx))
	cmd.AddCommand(NewGetCommand(ctx))

	return cmd
}
