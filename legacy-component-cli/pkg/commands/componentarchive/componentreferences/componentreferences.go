// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentreferences

import (
	"context"

	"github.com/spf13/cobra"
)

// NewCompRefCommand creates a new command to to modify component references of a component descriptor.
func NewCompRefCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "component-references",
		Aliases: []string{"component-reference", "compref", "ref"},
		Short:   "command to modify component references of a component descriptor",
	}
	cmd.AddCommand(NewAddCommand(ctx))
	return cmd
}
