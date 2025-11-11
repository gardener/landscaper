// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package signature

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
	"github.com/spf13/pflag"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/signatures"
)

type AddDigestsOptions struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// UploadBaseUrl is the base url where the digested component descriptor will be uploaded
	UploadBaseUrl string

	// Force to overwrite component descriptors on upload
	Force bool

	// Recursive to digest and upload all referenced component descriptors
	Recursive bool

	// SkipAccessTypes defines the access types that will be ignored for adding digests
	SkipAccessTypes []string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

func NewAddDigestsCommand(ctx context.Context) *cobra.Command {
	opts := &AddDigestsOptions{}
	cmd := &cobra.Command{
		Use:   "add-digests BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from an oci registry and add digests",
		Long: `
		fetch the component descriptor from an oci registry and add digests. optionally resolve and digest the referenced component descriptors.
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

func (o *AddDigestsOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	repoCtx := cdv2.NewOCIRegistryRepository(o.BaseUrl, "")

	ociClient, cache, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cdresolver := cdoci.NewResolver(ociClient)
	rootCd, blobResolver, err := cdresolver.ResolveWithBlobResolver(ctx, repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s:%s: %w", o.ComponentName, o.Version, err)
	}

	blobResolvers := map[string]ctf.BlobResolver{}
	blobResolvers[fmt.Sprintf("%s:%s", rootCd.Name, rootCd.Version)] = blobResolver

	skipAccessTypesMap := map[string]bool{}
	for _, v := range o.SkipAccessTypes {
		skipAccessTypesMap[v] = true
	}

	cds, err := signatures.RecursivelyAddDigestsToCd(rootCd, *repoCtx, ociClient, blobResolvers, context.TODO(), skipAccessTypesMap)
	if err != nil {
		return fmt.Errorf("unable to add digests to component descriptor: %w", err)
	}

	targetRepoCtx := cdv2.NewOCIRegistryRepository(o.UploadBaseUrl, "")

	if o.Recursive {
		for _, cd := range cds {
			logger.Log.Info(fmt.Sprintf("Uploading to %s %s %s", o.UploadBaseUrl, cd.Name, cd.Version))

			if err := signatures.UploadCDPreservingLocalOciBlobs(ctx, *cd, *targetRepoCtx, ociClient, cache, blobResolvers, o.Force, log); err != nil {
				return fmt.Errorf("unable to upload component descriptor %s:%s: %w", cd.Name, cd.Version, err)
			}
		}
	} else {
		if err := signatures.UploadCDPreservingLocalOciBlobs(ctx, *rootCd, *targetRepoCtx, ociClient, cache, blobResolvers, o.Force, log); err != nil {
			return fmt.Errorf("unable to upload component descriptor %s:%s: %w", rootCd.Name, rootCd.Version, err)
		}
	}

	return nil
}

func (o *AddDigestsOptions) Complete(args []string) error {
	o.BaseUrl = args[0]
	o.ComponentName = args[1]
	o.Version = args[2]

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return err
	}

	o.OciOptions.CacheDir = filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(o.OciOptions.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.OciOptions.CacheDir, err)
	}

	if len(o.BaseUrl) == 0 {
		return errors.New("a base url must be provided")
	}
	if len(o.ComponentName) == 0 {
		return errors.New("a component name must be provided")
	}
	if len(o.Version) == 0 {
		return errors.New("a component version must be provided")
	}
	if o.UploadBaseUrl == "" {
		return errors.New("a upload base url must be provided")
	}

	return nil
}

func (o *AddDigestsOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.UploadBaseUrl, "upload-base-url", "", "target repository context to upload the signed cd")
	fs.StringSliceVar(&o.SkipAccessTypes, "skip-access-types", []string{}, "comma separated list of access types that will not be digested")
	fs.BoolVar(&o.Force, "force", false, "force overwrite of already existing component descriptors")
	fs.BoolVar(&o.Recursive, "recursive", false, "recursively upload all referenced component descriptors")
	o.OciOptions.AddFlags(fs)
}
