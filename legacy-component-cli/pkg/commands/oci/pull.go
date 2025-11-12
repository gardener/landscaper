// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

const ConfigOutputName = "config"

type PullOptions struct {
	// Output defines the output directory or file where the artifact should be written to.
	// If a blob is defined, the output is the file where the data is written to.
	// If the whole artifact is downloaded a directory structure containing all blobs is created.
	Output string

	// Ref is the oci artifact reference.
	Ref string
	// BlobDigest defines the blob that should be downloaded.
	// If the digest is "config" automatically the config blob will be fetched.
	BlobDigest string

	// OCIOptions contains all oci client related options.
	OCIOptions ociopts.Options
}

func NewPullCommand(ctx context.Context) *cobra.Command {
	opts := &PullOptions{}
	cmd := &cobra.Command{
		Use:   "pull ARTIFACT_REFERENCE [config | blob digest]",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Pulls a oci artifact from a registry",
		Long: `
Pull downloads the specified oci artifact from a registry.
If no output directory is specified, the blob is written to stdout.

If a blob digest is given, the artifact will download the specific blob.
If no blob is given the whole artifact is downloaded and written to a directory.
If no output directory is specified, the artifact manifest is written to stdout.

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

func (o *PullOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.Output, "output-dir", "O", "", "specifies the output where the artifact should be written.")
	o.OCIOptions.AddFlags(fs)
}

func (o *PullOptions) Complete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one argument that defines the reference is needed")
	}
	o.Ref = args[0]

	if len(args) == 2 {
		o.BlobDigest = args[1]
	}

	return nil
}

func (o *PullOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	manifest, err := ociClient.GetManifest(ctx, o.Ref)
	if err != nil {
		return fmt.Errorf("unable to get manifest for %q: %w", o.Ref, err)
	}

	if len(o.BlobDigest) == 0 && len(o.Output) == 0 {
		// output manifest
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to serialize manifest: %w", err)
		}
		_, err = fmt.Fprint(os.Stdout, string(data))
		return err
	}

	if len(o.BlobDigest) != 0 {
		var desc *ocispecv1.Descriptor
		if o.BlobDigest == ConfigOutputName || o.BlobDigest == manifest.Config.Digest.String() {
			desc = &manifest.Config
		} else {
			desc = oci.GetLayerWithDigest(manifest.Layers, o.BlobDigest)
			if desc == nil {
				return fmt.Errorf("no layer in the manifest defined with digest %q", o.BlobDigest)
			}
		}

		if len(o.Output) == 0 {
			// output to stdout
			if err := ociClient.Fetch(ctx, o.Ref, *desc, os.Stdout); err != nil {
				return fmt.Errorf("unable to get blob %q from %q: %w", desc.Digest.String(), o.Ref, err)
			}
			return nil
		} else {
			if err := o.writeLayerToFile(ctx, ociClient, fs, o.Output, *desc); err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Successfully written file to %q", o.Output))
			return nil
		}

	}

	finfo, err := fs.Stat(o.Output)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unable to get info for output file %q: %w", o.Output, err)
		}
		if err := fs.MkdirAll(o.Output, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", o.Output, err)
		}
	} else {
		if !finfo.IsDir() {
			return fmt.Errorf("unable to write oci artifact as directory to file %q", o.Output)
		}
	}

	// write manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to serialize manifest: %w", err)
	}
	if err := vfs.WriteFile(fs, filepath.Join(o.Output, "manifest.json"), data, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write manifest: %w", err)
	}

	blobDir := filepath.Join(o.Output, "blobs")

	// write config
	if err := o.writeLayerToFile(ctx, ociClient, fs, filepath.Join(blobDir, "config"), manifest.Config); err != nil {
		return err
	}
	log.Info("Successfully written config")

	for _, layer := range manifest.Layers {
		if err := o.writeLayerToFile(ctx, ociClient, fs, filepath.Join(blobDir, string(layer.Digest.Algorithm()), layer.Digest.Encoded()), layer); err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Successfully written layer %q", layer.Digest.Encoded()))
	}

	return nil
}

func (o *PullOptions) writeLayerToFile(ctx context.Context, ociClient oci.Client, fs vfs.FileSystem, filename string, desc ocispecv1.Descriptor) error {
	finfo, err := fs.Stat(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unable to get info for output file %q: %w", filename, err)
		}
		if err := fs.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", filepath.Dir(filename), err)
		}
	} else {
		if finfo.IsDir() {
			return fmt.Errorf("unable to write blob to directoy %q", filename)
		}
	}
	file, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := ociClient.Fetch(ctx, o.Ref, desc, file); err != nil {
		return fmt.Errorf("unable to get blob %q from %q: %w", desc.Digest.String(), o.Ref, err)
	}
	return nil
}
