// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"context"
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
)

const defaultOutputPath = "./componentarchive"

// ExportOptions defines all options for the export command.
type ExportOptions struct {
	// ComponentArchivePath defines the path to the component archive
	ComponentArchivePath string
	// OutputPath defines the path where the exported component archive should be written to.
	OutputPath string
	// OutputFormat defines the output format of the component archive.
	OutputFormat ctf.ArchiveFormat
}

// NewExportCommand creates a new export command that packages a component archive and
// exports is as tar or compressed tar.
func NewExportCommand(ctx context.Context) *cobra.Command {
	opts := &ExportOptions{}
	cmd := &cobra.Command{
		Use:   "export COMPONENT_ARCHIVE_PATH [-o output-dir/file] [-f {fs|tar|tgz}]",
		Args:  cobra.ExactArgs(1),
		Short: "Exports a component archive as defined by CTF",
		Long: `
Export command exports a component archive as defined by CTF (CNUDIE Transport Format).
If the given component-archive path points to a directory, the archive is expected to be a extracted component-archive on the filesystem.
Then it is exported as tar or optionally as compressed tar.

If the given path points to a file, the archive is read as tar or compressed tar (tar.gz) and exported as filesystem to the given location.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			if err := opts.Run(ctx, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			fmt.Printf("Successfully exported component archive to %s\n", opts.OutputPath)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the export for a component archive.
func (o *ExportOptions) Run(_ context.Context, fs vfs.FileSystem) error {
	ca, format, err := componentarchive.Parse(fs, o.ComponentArchivePath)
	if err != nil {
		return err
	}
	if format == ctf.ArchiveFormatFilesystem {
		return o.export(fs, ca, ctf.ArchiveFormatTar)
	} else {
		return o.export(fs, ca, ctf.ArchiveFormatFilesystem)
	}
}

func (o *ExportOptions) export(fs vfs.FileSystem, ca *ctf.ComponentArchive, defaultFormat ctf.ArchiveFormat) error {
	if len(o.OutputFormat) == 0 {
		o.OutputFormat = defaultFormat
	}

	return componentarchive.Write(fs, o.OutputPath, ca, o.OutputFormat)
}

// Complete parses the given command arguments and applies default options.
func (o *ExportOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one argument that contains the path to the component archive")
	}
	o.ComponentArchivePath = args[0]

	if len(o.OutputPath) == 0 {
		o.OutputPath = defaultOutputPath
	}

	return o.validate()
}

func (o *ExportOptions) validate() error {
	return componentarchive.ValidateOutputFormat(o.OutputFormat, true)
}

func (o *ExportOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.OutputPath, "out", "o", "", "writes the resulting archive to the given path")
	componentarchive.OutputFormatVar(fs, &o.OutputFormat, "format", "", componentarchive.DefaultOutputFormatUsage)
}
