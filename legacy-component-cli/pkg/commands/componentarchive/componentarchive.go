// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	pflag "github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/componentreferences"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/remote"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/resources"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/signature"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/sources"
	ctfcmd "github.com/gardener/landscaper/legacy-component-cli/pkg/commands/ctf"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
)

type ComponentArchiveOptions struct {
	componentarchive.BuilderOptions
	TemplateOptions template.Options

	// ResourcesPaths defines all paths to the resource definitions.
	ResourcesPaths []string
	// SourcesPaths defines all paths to the source definitions.
	SourcesPaths []string
	// ComponentReferencesPaths defines all paths to the component-references definitions.
	ComponentReferencesPaths []string
	// ArchiveFormat defines the component archive format of a component archive defines in a filesystem
	ArchiveFormat ctf.ArchiveFormat
	// TempDir is the temporary directory where the component archive is build.
	// Optional will be defaulted to a random os-specific temporary directory
	TempDir string

	// CTFPath is the path to the ctf.
	// The component archive will be automatically added to the ctf if set
	CTFPath string
}

// NewComponentArchiveCommand creates a new component archive command.
func NewComponentArchiveCommand(ctx context.Context) *cobra.Command {
	opts := &ComponentArchiveOptions{}
	cmd := &cobra.Command{
		Use:     "component-archive [component-archive-path] [ctf-path]",
		Aliases: []string{"componentarchive", "ca", "archive"},
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
	cmd.AddCommand(NewCreateCommand(ctx))
	cmd.AddCommand(NewExportCommand(ctx))
	cmd.AddCommand(remote.NewRemoteCommand(ctx))
	cmd.AddCommand(resources.NewResourcesCommand(ctx))
	cmd.AddCommand(componentreferences.NewCompRefCommand(ctx))
	cmd.AddCommand(sources.NewSourcesCommand(ctx))
	cmd.AddCommand(signature.NewSignaturesCommand(ctx))
	return cmd
}

func (o *ComponentArchiveOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	if len(o.TempDir) == 0 {
		var err error
		o.TempDir, err = vfs.TempDir(fs, fs.FSTempDir(), "ca-")
		if err != nil {
			return fmt.Errorf("unable to create temporary directory: %w", err)
		}
		defer func() {
			if err := fs.RemoveAll(o.TempDir); err != nil {
				log.Error(err, "unable to remove temporary directory %q: %w", o.TempDir, err)
			}
		}()
		log.V(5).Info("using temporary directory", "dir", o.TempDir)
	}

	// ensure that a optional archive exists
	_, err := o.Build(fs)
	if err != nil {
		return err
	}

	// if no resource, sources or component references are added
	// directly add the given component archive to the ctf
	if len(o.ResourcesPaths) == 0 && len(o.SourcesPaths) == 0 && len(o.ComponentReferencesPaths) == 0 {
		ctfAdd := &ctfcmd.AddOptions{
			CTFPath:           o.CTFPath,
			ArchiveFormat:     o.ArchiveFormat,
			ComponentArchives: []string{o.ComponentArchivePath},
		}
		if err := ctfAdd.Run(ctx, log, fs); err != nil {
			return fmt.Errorf("unable to add component archive to ctf: %w", err)
		}
		log.Info("Successfully added ctf\n")
	}

	// only copy essential files to the temp dir
	log.V(5).Info("copy component descriptor to temp dir")
	if err := o.copyToTempDir(fs); err != nil {
		return err
	}
	o.ComponentArchivePath = o.TempDir

	if len(o.ResourcesPaths) != 0 {
		add := &resources.Options{
			BuilderOptions:      o.BuilderOptions,
			TemplateOptions:     o.TemplateOptions,
			ResourceObjectPaths: o.ResourcesPaths,
		}
		if err := add.Run(ctx, log, fs); err != nil {
			return err
		}
	}
	if len(o.SourcesPaths) != 0 {
		add := &sources.Options{
			BuilderOptions:    o.BuilderOptions,
			TemplateOptions:   o.TemplateOptions,
			SourceObjectPaths: o.SourcesPaths,
		}
		if err := add.Run(ctx, log, fs); err != nil {
			return err
		}
	}
	if len(o.ComponentReferencesPaths) != 0 {
		add := &componentreferences.Options{
			BuilderOptions:                o.BuilderOptions,
			TemplateOptions:               o.TemplateOptions,
			ComponentReferenceObjectPaths: o.ComponentReferencesPaths,
		}
		if err := add.Run(ctx, log, fs); err != nil {
			return err
		}
	}

	ctfAdd := &ctfcmd.AddOptions{
		CTFPath:           o.CTFPath,
		ArchiveFormat:     o.ArchiveFormat,
		ComponentArchives: []string{o.TempDir},
	}
	if err := ctfAdd.Run(ctx, log, fs); err != nil {
		return fmt.Errorf("unable to add component archive to ctf: %w", err)
	}
	log.Info("Successfully added ctf\n")
	return nil
}

func (o *ComponentArchiveOptions) copyToTempDir(fs vfs.FileSystem) error {
	// only copy essential files to the temp dir
	srcCompDescFilePath := filepath.Join(o.ComponentArchivePath, ctf.ComponentDescriptorFileName)
	dstCompDescFilePath := filepath.Join(o.TempDir, ctf.ComponentDescriptorFileName)
	if err := vfs.CopyFile(fs, srcCompDescFilePath, fs, dstCompDescFilePath); err != nil {
		return fmt.Errorf("unable to copy component descriptor to %q: %w", dstCompDescFilePath, err)
	}
	srcBlobDir := filepath.Join(o.ComponentArchivePath, ctf.BlobsDirectoryName)
	dstBlobDir := filepath.Join(o.TempDir, ctf.BlobsDirectoryName)
	info, err := fs.Stat(srcBlobDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is expected to be a directory", srcBlobDir)
	}
	if err := vfs.CopyDir(fs, srcBlobDir, fs, dstBlobDir); err != nil {
		return fmt.Errorf("unable to copy blob directory to %q: %w", dstBlobDir, err)
	}
	return nil
}

func (o *ComponentArchiveOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&o.ResourcesPaths, "resources", "r", []string{}, "path to resources definition")
	fs.StringArrayVarP(&o.SourcesPaths, "sources", "s", []string{}, "path to sources definition")
	fs.StringArrayVarP(&o.ComponentReferencesPaths, "component-ref", "c", []string{}, "path to component references definition")
	componentarchive.OutputFormatVar(fs, &o.ArchiveFormat, "format", ctf.ArchiveFormatTar,
		componentarchive.ArchiveOutputFormatUsage)
	fs.StringVar(&o.TempDir, "temp-dir", "", "temporary directory where the component archive is build. Defaults to a os-specific temp dir")
	o.BuilderOptions.AddFlags(fs)
}

func (o *ComponentArchiveOptions) Complete(args []string) error {
	args = o.TemplateOptions.Parse(args)

	if len(args) == 0 {
		return errors.New("at least a component archive path argument has to be defined")
	}
	o.ComponentArchivePath = args[0]
	o.Default()

	if len(args) == 2 {
		o.CTFPath = args[1]
	} else {
		// todo make ctf optional
		return errors.New("currently a ctf path is required")
	}
	return o.validate()
}

func (o *ComponentArchiveOptions) validate() error {
	return o.Validate()
}
