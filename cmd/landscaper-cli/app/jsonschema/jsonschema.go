// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"context"

	"github.com/spf13/cobra"
)

// NewComponentsCommand creates a new components command.
func NewComponentsCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "jsonschema",
		Aliases: []string{"js"},
		Short:   "command to interact with a jsonschema stored in an oci registry",
	}

	cmd.AddCommand(NewPushCommand(ctx))
	cmd.AddCommand(NewGetCommand(ctx))

	return cmd
}
