// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
	"context"

	"github.com/spf13/cobra"
)

// NewImageVectorCommand creates a new command to to modify component references of a component descriptor.
func NewImageVectorCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "image-vector",
		Aliases: []string{"imagevector", "iv"},
		Short:   "command to add resource from a image vector and retrieve from a component descriptor",
	}
	cmd.AddCommand(NewAddCommand(ctx))
	cmd.AddCommand(NewGenerateOverwriteCommand(ctx))
	return cmd
}
