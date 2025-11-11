// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"

	"github.com/spf13/cobra"
)

// NewRemoteCommand creates a new command to interact with remote component descriptors.
func NewRemoteCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "command to interact with component descriptors stored in an oci registry",
	}

	cmd.AddCommand(NewPushCommand(ctx))
	cmd.AddCommand(NewGetCommand(ctx))
	cmd.AddCommand(NewCopyCommand(ctx))

	return cmd
}
