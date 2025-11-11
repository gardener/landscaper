// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"context"

	"github.com/spf13/cobra"
)

// NewCTFCommand creates a new ctf command.
func NewCTFCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "ctf",
	}
	cmd.AddCommand(NewPushCommand(ctx))
	cmd.AddCommand(NewAddCommand(ctx))
	return cmd
}
