// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

type AddOptions struct {
	// CTFPath is the path to the directory containing the ctf archive.
	CTFPath string
	// ArchiveFormat defines the component archive format of a component archive defines in a filesystem
	ArchiveFormat ctf.ArchiveFormat

	ComponentArchives []string
}

// NewAddCommand creates a new definition command to push definitions
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &AddOptions{}
	cmd := &cobra.Command{
		Use:   "add CTF_PATH [-f component-archive]...",
		Args:  cobra.RangeArgs(1, 4),
		Short: "Adds component archives to a ctf",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			fmt.Print("Successfully added ctf\n")
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *AddOptions) Run(_ context.Context, log logr.Logger, fs vfs.FileSystem) error {
	info, err := fs.Stat(o.CTFPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unable to get info for %s: %w", o.CTFPath, err)
		}
		log.Info("CTF Archive does not exist creating a new one")

		file, err := fs.OpenFile(o.CTFPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open file for %s: %w", o.CTFPath, err)
		}
		tw := tar.NewWriter(file)
		if err := tw.Close(); err != nil {
			return fmt.Errorf("unable to close tarwriter for emtpy tar: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("unable to close tarwriter for emtpy tar: %w", err)
		}
		info, err = fs.Stat(o.CTFPath)
		if err != nil {
			return fmt.Errorf("unable to get info for %s: %w", o.CTFPath, err)
		}
	}
	if info.IsDir() {
		return fmt.Errorf(`%q is a directory. 
It is expected that the given path points to a CTF Archive`, o.CTFPath)
	}

	ctfArchive, err := ctf.NewCTF(fs, o.CTFPath)
	if err != nil {
		return fmt.Errorf("unable to open ctf at %q: %s", o.CTFPath, err.Error())
	}

	for _, caPath := range o.ComponentArchives {
		ca, _, err := componentarchive.Parse(fs, caPath)
		if err != nil {
			return err
		}
		if err := ctfArchive.AddComponentArchiveWithName(
			utils.CTFComponentArchiveFilename(ca.ComponentDescriptor.GetName(),
				ca.ComponentDescriptor.GetVersion()),
			ca,
			o.ArchiveFormat,
		); err != nil {
			return fmt.Errorf("unable to add component archive %q to ctf: %s", ca.ComponentDescriptor.GetName(), err.Error())
		}
		log.Info(fmt.Sprintf("Successfully added component archive from %q", caPath))
	}
	if err := ctfArchive.Write(); err != nil {
		return fmt.Errorf("unable to write modified ctf archive: %s", err.Error())
	}
	return ctfArchive.Close()
}

func (o *AddOptions) Complete(args []string) error {
	o.CTFPath = args[0]

	if err := o.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates push options
func (o *AddOptions) Validate() error {
	if len(o.CTFPath) == 0 {
		return errors.New("a path to the component descriptor must be provided")
	}

	if len(o.ComponentArchives) == 0 {
		return errors.New("no archives to add")
	}

	if o.ArchiveFormat != ctf.ArchiveFormatTar &&
		o.ArchiveFormat != ctf.ArchiveFormatTarGzip {
		return fmt.Errorf("unsupported archive format %q", o.ArchiveFormat)
	}
	return nil
}

func (o *AddOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&o.ComponentArchives, "component-archive", "f", []string{},
		"path to the component archives to be added. Note that the component archives have to be tar archives.")
	componentarchive.OutputFormatVar(fs, &o.ArchiveFormat, "format", ctf.ArchiveFormatTar,
		componentarchive.ArchiveOutputFormatUsage)
}
