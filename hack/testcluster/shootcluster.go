// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/spf13/cobra"
)

func NewShootClusterCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shootcluster",
		Short:   "",
		Long:    "",
		Example: "",
	}
	cmd.AddCommand(NewCreateShootClusterCommand(ctx))
	cmd.AddCommand(NewDeleteShootClusterCommand(ctx))
	return cmd
}
