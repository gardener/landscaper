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

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

// CopyOptions defines all options that are used
type CopyOptions struct {
	// SourceRef is the source oci artifact reference.
	SourceRef string
	// TargetRef is the target oci artifact reference where the artifact is copied to.
	TargetRef string

	// OCIOptions contains all oci client related options.
	OCIOptions ociopts.Options
}

func NewCopyCommand(ctx context.Context) *cobra.Command {
	opts := &CopyOptions{}
	cmd := &cobra.Command{
		Use:   "copy SOURCE_ARTIFACT_REFERENCE TARGET_ARTIFACT_REFERENCE",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Copies a oci artifact from a registry to another",
		Long: `
Copy copies a artifact from a source to a target registry.
The artifact is copied without modification.
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

func (o *CopyOptions) AddFlags(fs *pflag.FlagSet) {
	o.OCIOptions.AddFlags(fs)
}

func (o *CopyOptions) Complete(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("a source and target oci artifact ref")
	}
	o.SourceRef = args[0]
	o.TargetRef = args[1]
	return nil
}

func (o *CopyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}
	if err := ociclient.Copy(ctx, ociClient, o.SourceRef, o.TargetRef); err != nil {
		return err
	}
	fmt.Printf("Successfully copied %q to %q", o.SourceRef, o.TargetRef)
	return nil
}
