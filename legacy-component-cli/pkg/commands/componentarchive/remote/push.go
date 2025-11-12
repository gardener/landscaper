// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

// PushOptions contains all options to upload a component archive.
type PushOptions struct {
	// AdditionalTags defines additional tags that the oci artifact should be tagged with.
	AdditionalTags []string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
	// BuilderOptions for the component archive builder
	componentarchive.BuilderOptions
}

// NewPushCommand creates a new definition command to push definitions
func NewPushCommand(ctx context.Context) *cobra.Command {
	opts := &PushOptions{}
	cmd := &cobra.Command{
		Use:   "push COMPONENT_DESCRIPTOR_PATH",
		Args:  cobra.RangeArgs(1, 4),
		Short: "pushes a component archive to an oci repository",
		Long: `
pushes a component archive with the component descriptor and its local blobs to an oci repository.

The command can be called in 2 different ways:

push [path to component descriptor]
- The cli will read all necessary parameters from the component descriptor.

push [baseurl] [componentname] [Version] [path to component descriptor]
- The cli will add the baseurl as repository context and validate the name and Version.
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

func (o *PushOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, cache, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	archive, err := o.Build(fs)
	if err != nil {
		return fmt.Errorf("unable to build component archive: %w", err)
	}
	// update repository context
	if len(o.BaseUrl) != 0 {
		if err := cdv2.InjectRepositoryContext(archive.ComponentDescriptor, cdv2.NewOCIRegistryRepository(o.BaseUrl, "")); err != nil {
			return fmt.Errorf("unable to add repository context to component descriptor: %w", err)
		}
	}

	manifest, err := cdoci.NewManifestBuilder(cache, archive).Build(ctx)
	if err != nil {
		return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
	}

	ref, err := components.OCIRef(archive.ComponentDescriptor.GetEffectiveRepositoryContext(), archive.ComponentDescriptor.Name, archive.ComponentDescriptor.Version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}
	if err := ociClient.PushManifest(ctx, ref, manifest); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Successfully uploaded component descriptor at %q", ref))

	for _, tag := range o.AdditionalTags {
		ref, err := components.OCIRef(archive.ComponentDescriptor.GetEffectiveRepositoryContext(), archive.ComponentDescriptor.Name, tag)
		if err != nil {
			return fmt.Errorf("invalid component reference: %w", err)
		}
		if err := ociClient.PushManifest(ctx, ref, manifest); err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Successfully tagged component descriptor %q", ref))
	}
	return nil
}

func (o *PushOptions) Complete(args []string) error {
	switch len(args) {
	case 1:
		o.ComponentArchivePath = args[0]
	case 4:
		o.BaseUrl = args[0]
		o.Name = args[1]
		o.Version = args[2]
		o.ComponentArchivePath = args[3]
	}

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
	// todo: validate references exist
	return o.BuilderOptions.Validate()
}

func (o *PushOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&o.AdditionalTags, "tag", "t", []string{}, "set additional tags on the oci artifact")
	o.OciOptions.AddFlags(fs)
	o.BuilderOptions.AddFlags(fs)
}
