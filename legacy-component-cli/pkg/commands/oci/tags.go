// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

type TagsOptions struct {
	// Ref is the oci artifact reference.
	Ref string

	// OCIOptions contains all oci client related options.
	OCIOptions ociopts.Options
}

func NewTagsCommand(ctx context.Context) *cobra.Command {
	opts := &TagsOptions{}
	cmd := &cobra.Command{
		Use:   "tags ARTIFACT_REFERENCE",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Lists all tags of artifact reference",
		Long: `
tags lists all tags for a specific artifact reference that is known by the registry.

`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

func (o *TagsOptions) AddFlags(fs *pflag.FlagSet) {
	o.OCIOptions.AddFlags(fs)
}

func (o *TagsOptions) Complete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one argument that defines the reference is needed")
	}
	o.Ref = args[0]
	return nil
}

func (o *TagsOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	tags, err := ociClient.ListTags(ctx, o.Ref)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		fmt.Println(tag)
	}
	return nil
}
