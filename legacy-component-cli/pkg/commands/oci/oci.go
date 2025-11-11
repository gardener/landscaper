// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"

	"github.com/spf13/cobra"
)

// NewOCICommand creates a new ctf command.
func NewOCICommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "oci",
	}
	cmd.AddCommand(NewPullCommand(ctx))
	cmd.AddCommand(NewCopyCommand(ctx))
	cmd.AddCommand(NewTagsCommand(ctx))
	cmd.AddCommand(NewRepositoriesCommand(ctx))
	return cmd
}
