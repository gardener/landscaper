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
	"github.com/gardener/landscaper/test/utils"
)

func NewDeleteShootClusterCommand(ctx context.Context) *cobra.Command {
	opts := &DeleteShootClusterOptions{}
	cmd := &cobra.Command{
		Use:          "delete",
		Short:        "deletes a shoot cluster",
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

type DeleteShootClusterOptions struct {
	GardenClusterKubeconfigPath string
	Namespace                   string
	AuthDirectoryPath           string
}

// AddFlags adds flags for the options to a flagset
func (o *DeleteShootClusterOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.StringVar(&o.GardenClusterKubeconfigPath, "kubeconfig", "", "the path to the kubeconfig of the garden cluster")
	fs.StringVarP(&o.Namespace, "namespace", "n", "", "namespace of the cluster that should be deleted")
	fs.StringVar(&o.AuthDirectoryPath, "cluster-auth", "", "path to a directory that must contain a file 'clustername' containing the name of the cluster to be deleted")
}

func (o *DeleteShootClusterOptions) Complete() error {
	if err := o.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *DeleteShootClusterOptions) Validate() error {
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

func (o *DeleteShootClusterOptions) Run(ctx context.Context) error {
	log := utils.NewLogger().WithTimestamp()

	shootClusterManager, err := pkg.NewShootClusterManager(log, o.GardenClusterKubeconfigPath, o.Namespace, o.AuthDirectoryPath,
		0, 0, "1h")

	if err != nil {
		return err
	}

	if err := shootClusterManager.DeleteShootCluster(ctx); err != nil {
		return err
	}

	return nil
}
