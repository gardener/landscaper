// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

// CreateOptions defines all options for the create command.
type CreateOptions struct {
	componentarchive.BuilderOptions
}

// NewCreateCommand creates a new component descriptor
func NewCreateCommand(ctx context.Context) *cobra.Command {
	opts := &CreateOptions{}
	cmd := &cobra.Command{
		Use:   "create COMPONENT_ARCHIVE_PATH",
		Args:  cobra.ExactArgs(1),
		Short: "Creates a component archive with a component descriptor",
		Long: `
Create command creates a new component archive directory with a "component-descriptor.yaml" file.
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
			fmt.Printf("Successfully created component archive at %s\n", args[0])
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the export for a component archive.
func (o *CreateOptions) Run(_ context.Context, log logr.Logger, fs vfs.FileSystem) error {
	if o.Overwrite {
		log.V(3).Info("overwrite enabled")
	}
	_, err := o.BuilderOptions.Build(fs)
	return err
}

// Complete parses the given command arguments and applies default options.
func (o *CreateOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one argument that contains the path to the component archive")
	}
	o.ComponentArchivePath = args[0]

	if len(o.Name) == 0 {
		return errors.New("a name has to be provided for a minimal component descriptor")
	}

	if len(o.Version) == 0 {
		return errors.New("a version has to be provided for a minimal component descriptor")
	}

	return o.validate()
}

func (o *CreateOptions) validate() error {
	return o.BuilderOptions.Validate()
}

func (o *CreateOptions) AddFlags(fs *pflag.FlagSet) {
	o.BuilderOptions.AddFlags(fs)
	fs.BoolVarP(&o.BuilderOptions.Overwrite, "overwrite", "w", false, "overwrites the existing component")
}
