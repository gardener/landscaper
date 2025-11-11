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
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/signature/verify"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

type CheckDigestsOptions struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

func NewCheckDigest(ctx context.Context) *cobra.Command {
	opts := &CheckDigestsOptions{}
	cmd := &cobra.Command{
		Use:   "check-digests BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from an oci registry and check digests",
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

func (o *CheckDigestsOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	repoCtx := cdv2.NewOCIRegistryRepository(o.BaseUrl, "")

	ociClient, _, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cdresolver := cdoci.NewResolver(ociClient)
	cd, err := cdresolver.Resolve(ctx, repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s:%s: %w", o.ComponentName, o.Version, err)
	}

	// check componentReferences and resources
	if err := verify.CheckCdDigests(cd, *repoCtx, ociClient, context.TODO()); err != nil {
		return fmt.Errorf("unable to check component descriptor digests: %w", err)
	}

	return nil
}

// Complete validates the arguments and flags from the command line
func (o *CheckDigestsOptions) Complete(args []string) error {
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
	return nil
}

func (o *CheckDigestsOptions) AddFlags(fs *pflag.FlagSet) {
	o.OciOptions.AddFlags(fs)
}
