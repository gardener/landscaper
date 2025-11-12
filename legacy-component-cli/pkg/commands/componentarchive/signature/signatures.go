// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package signature

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/signature/sign"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/signature/verify"
)

// NewSignaturesCommand creates a new command to interact with signatures.
func NewSignaturesCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "signatures",
		Aliases: []string{"signature"},
		Short:   "command to work with signatures and digests in component descriptors",
	}

	cmd.AddCommand(NewAddDigestsCommand(ctx))
	cmd.AddCommand(NewCheckDigest(ctx))
	cmd.AddCommand(sign.NewSignCommand(ctx))
	cmd.AddCommand(verify.NewVerifyCommand(ctx))

	return cmd
}
