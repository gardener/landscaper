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

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

type RepositoriesOptions struct {
	// Registry is url of the host.
	Registry string

	// OCIOptions contains all oci client related options.
	OCIOptions ociopts.Options
}

func NewRepositoriesCommand(ctx context.Context) *cobra.Command {
	opts := &RepositoriesOptions{}
	cmd := &cobra.Command{
		Use:     "repositories REPOSITORY_PREFIX",
		Aliases: []string{"repos", "repo"},
		Args:    cobra.RangeArgs(1, 2),
		Short:   "Lists all repositories of the registry",
		Long: `
repositories lists all known repositories of the registry.

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

func (o *RepositoriesOptions) AddFlags(fs *pflag.FlagSet) {
	o.OCIOptions.AddFlags(fs)
}

func (o *RepositoriesOptions) Complete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one argument that defines the reference is needed")
	}
	o.Registry = args[0]
	return nil
}

func (o *RepositoriesOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	repos, err := ociClient.ListRepositories(ctx, o.Registry)
	if err != nil {
		return err
	}
	for _, repo := range repos {
		fmt.Println(repo)
	}
	return nil
}
