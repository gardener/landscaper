// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

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

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

type PushOptions struct {
	// CTFPath is the path to the directory containing the ctf archive.
	CTFPath string
	// BaseUrl is the repository context base url for all included component descriptors.
	BaseUrl string
	// AdditionalTags defines additional tags that the oci artifact should be tagged with.
	AdditionalTags []string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewPushCommand creates a new definition command to push definitions
func NewPushCommand(ctx context.Context) *cobra.Command {
	opts := &PushOptions{}
	cmd := &cobra.Command{
		Use:   "push CTF_PATH",
		Args:  cobra.ExactArgs(1),
		Short: "Pushes all archives of a ctf to a remote repository",
		Long: `
Push pushes all component archives and oci artifacts to the defined oci repository.

The oci repository is automatically determined based on the component/artifact descriptor (repositoryContext, component name and version).

Note: Currently only component archives are supoprted. Generic OCI Artifacts will be supported in the future.
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

			fmt.Print("Successfully uploaded ctf\n")
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *PushOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	info, err := fs.Stat(o.CTFPath)
	if err != nil {
		return fmt.Errorf("unable to get info for %s: %w", o.CTFPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf(`%q is a directory. 
It is expected that the given path points to a CTF Archive`, o.CTFPath)
	}

	ociClient, cache, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	ctfArchive, err := ctf.NewCTF(fs, o.CTFPath)
	if err != nil {
		return fmt.Errorf("unable to open ctf at %q: %s", o.CTFPath, err.Error())
	}

	err = ctfArchive.Walk(func(ca *ctf.ComponentArchive) error {
		// update repository context
		if len(o.BaseUrl) != 0 {
			if err := cdv2.InjectRepositoryContext(ca.ComponentDescriptor, cdv2.NewOCIRegistryRepository(o.BaseUrl, "")); err != nil {
				return fmt.Errorf("unable to add repository context: %w", err)
			}
		}

		manifest, err := cdoci.NewManifestBuilder(cache, ca).Build(ctx)
		if err != nil {
			return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
		}

		ref, err := components.OCIRef(ca.ComponentDescriptor.GetEffectiveRepositoryContext(), ca.ComponentDescriptor.GetName(), ca.ComponentDescriptor.GetVersion())
		if err != nil {
			return fmt.Errorf("unable to calculate oci ref for %q: %s", ca.ComponentDescriptor.GetName(), err.Error())
		}
		if err := ociClient.PushManifest(ctx, ref, manifest); err != nil {
			return fmt.Errorf("unable to upload component archive to %q: %s", ref, err.Error())
		}
		log.Info(fmt.Sprintf("Successfully uploaded component archive to %q", ref))

		for _, tag := range o.AdditionalTags {
			ref, err := components.OCIRef(ca.ComponentDescriptor.GetEffectiveRepositoryContext(), ca.ComponentDescriptor.GetName(), tag)
			if err != nil {
				return fmt.Errorf("unable to calculate oci ref for %q: %s", ca.ComponentDescriptor.GetName(), err.Error())
			}
			if err := ociClient.PushManifest(ctx, ref, manifest); err != nil {
				return fmt.Errorf("unable to upload component archive to %q: %s", ref, err.Error())
			}
			log.Info(fmt.Sprintf("Successfully tagged component archive with %q", ref))
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error while reading component archives in ctf: %w", err)
	}

	return ctfArchive.Close()
}

func (o *PushOptions) Complete(args []string) error {
	o.CTFPath = args[0]

	var err error
	o.OciOptions.CacheDir, err = utils.CacheDir()
	if err != nil {
		return fmt.Errorf("unable to get oci cache directory: %w", err)
	}

	if err := o.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates push options
func (o *PushOptions) Validate() error {
	if len(o.CTFPath) == 0 {
		return errors.New("a path to the component descriptor must be provided")
	}
	return nil
}

func (o *PushOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.BaseUrl, "repo-ctx", "", "repository context url for component to upload. The repository url will be automatically added to the repository contexts.")
	fs.StringArrayVarP(&o.AdditionalTags, "tag", "t", []string{}, "set additional tags on the oci artifact")

	o.OciOptions.AddFlags(fs)
}
