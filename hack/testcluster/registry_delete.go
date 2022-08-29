// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/hack/testcluster/pkg"
)

// NewDeleteRegistryCommand creates a new delete registry command.
func NewDeleteRegistryCommand(ctx context.Context) *cobra.Command {
	opts := &DeleteRegistryOptions{}
	cmd := &cobra.Command{
		Use:          "delete",
		Short:        "deletes a previously created test registry",
		Long:         "",
		Example:      "",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(); err != nil {
				return err
			}
			return opts.Run(ctx)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// DeleteRegistryOptions defines all options that are needed for delete registry command.
type DeleteRegistryOptions struct {
	CommonOptions
}

// AddFlags adds flags for the options to a flagset
func (o *DeleteRegistryOptions) AddFlags(fs *pflag.FlagSet) {
	o.CommonOptions.AddFlags(fs)
}

func (o *DeleteRegistryOptions) Complete() error {
	if err := o.Validate(); err != nil {
		return err
	}
	return o.CommonOptions.Complete()
}

func (o *DeleteRegistryOptions) Validate() error {
	return o.CommonOptions.Validate()
}

func (o *DeleteRegistryOptions) Run(ctx context.Context) error {
	logger := utils.NewLogger().WithTimestamp()
	return pkg.DeleteRegistry(ctx,
		logger,
		o.kubeClient,
		o.Namespace,
		o.ID,
		o.Timeout)
}
