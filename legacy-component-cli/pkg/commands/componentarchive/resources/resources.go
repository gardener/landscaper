// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	"github.com/spf13/cobra"
)

// NewResourcesCommand creates a new command to to modify resources of a component descriptor.
func NewResourcesCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resources",
		Aliases: []string{"resource", "res"},
		Short:   "command to modify resources of a component descriptor",
	}
	cmd.AddCommand(NewAddCommand(ctx))
	return cmd
}
