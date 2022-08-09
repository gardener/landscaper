// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/hack/testcluster/pkg"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
)

func NewCreateShootClusterCommand(ctx context.Context) *cobra.Command {
	opts := &CreateShootClusterOptions{}
	cmd := &cobra.Command{
		Use:          "create",
		Short:        "creates a new shoot cluster",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(); err != nil {
				return err
			}
			return opts.Run(ctx)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// CreateShootClusterOptions defines all options that are needed for create registry command.
type CreateShootClusterOptions struct {
	GardenClusterKubeconfigPath string
	Namespace                   string
	AuthDirectoryPath           string
}

// AddFlags adds flags for the options to a flagset
func (o *CreateShootClusterOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.StringVar(&o.GardenClusterKubeconfigPath, "kubeconfig", "", "the path to the kubeconfig of the garden cluster")
	fs.StringVarP(&o.Namespace, "namespace", "n", "", "namespace where the cluster should be created")
	fs.StringVar(&o.AuthDirectoryPath, "cluster-auth", "", "the path to the auth directory")
}

func (o *CreateShootClusterOptions) Complete() error {
	if err := o.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *CreateShootClusterOptions) Validate() error {
	if o.GardenClusterKubeconfigPath == "" {
		return errors.New("no path to gardener kubeconfig specified")
	}

	if o.Namespace == "" {
		return errors.New("no namespace specified")
	}

	if o.AuthDirectoryPath == "" {
		return errors.New("no path to an auth directory specified (the directory to which name and kubeconfig " +
			"of the test cluster will be exported)")
	}

	return nil
}

func (o *CreateShootClusterOptions) Run(ctx context.Context) error {
	log := simplelogger.NewLogger().WithTimestamp()

	shootClusterManager := pkg.NewShootClusterManager(log, o.GardenClusterKubeconfigPath, o.Namespace, o.AuthDirectoryPath)

	if err := shootClusterManager.CreateShootCluster(ctx); err != nil {
		return err
	}

	return nil
}
