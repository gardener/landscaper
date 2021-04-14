// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/spf13/cobra"
)

func NewClusterCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "",
		Long:    "",
		Example: "",
	}
	cmd.AddCommand(NewCreateClusterCommand(ctx))
	cmd.AddCommand(NewDeleteClusterCommand(ctx))
	return cmd
}
