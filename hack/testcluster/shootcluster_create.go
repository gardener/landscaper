// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/hack/testcluster/pkg"
	"github.com/gardener/landscaper/test/utils"
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
	GardenClusterKubeconfigPath  string
	Namespace                    string
	AuthDirectoryPath            string
	MaxNumOfClusters             int
	NumClustersStartDeleteOldest int
	DurationForClusterDeletion   string
}

// AddFlags adds flags for the options to a flagset
func (o *CreateShootClusterOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.StringVar(&o.GardenClusterKubeconfigPath, "kubeconfig", "", "the path to the kubeconfig of the garden cluster")
	fs.StringVarP(&o.Namespace, "namespace", "n", "", "namespace where the cluster should be created")
	fs.StringVar(&o.AuthDirectoryPath, "cluster-auth", "", "the path to the auth directory")
	fs.IntVar(&o.MaxNumOfClusters, "max-num-cluster", 15, "maximal number of clusters")
	fs.IntVar(&o.NumClustersStartDeleteOldest, "num-clusters-start-delete-oldest", 10, "number of clusters to start deletion of the oldest")
	fs.StringVar(&o.DurationForClusterDeletion, "duration-for-cluster-deletion", "48h", "test cluster existing longer than this will be deleted")
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

	if o.MaxNumOfClusters < 1 {
		return errors.New("maximal number of clusters is lower than one")
	}

	if o.NumClustersStartDeleteOldest < 1 || o.NumClustersStartDeleteOldest >= o.MaxNumOfClusters {
		return errors.New("number of cluster to start delete oldest clusters is lower than one or larger or equal than maximal number of clusters")
	}

	_, err := time.ParseDuration(o.DurationForClusterDeletion)
	if err != nil {
		return errors.New("duration for cluster deletion has wrong format: " + o.DurationForClusterDeletion)
	}

	if o.AuthDirectoryPath == "" {
		return errors.New("no path to an auth directory specified (the directory to which name and kubeconfig " +
			"of the test cluster will be exported)")
	}

	return nil
}

func (o *CreateShootClusterOptions) Run(ctx context.Context) error {
	log := utils.NewLogger().WithTimestamp()

	log.Logfln("Create cluster with:")
	log.Logfln("  GardenClusterKubeconfigPath: " + o.GardenClusterKubeconfigPath)
	log.Logfln("  Namespace: " + o.Namespace)
	log.Logfln("  AuthDirectoryPath: " + o.AuthDirectoryPath)
	log.Logfln("  MaxNumOfClusters: " + strconv.Itoa(o.MaxNumOfClusters))
	log.Logfln("  NumClustersStartDeleteOldest: " + strconv.Itoa(o.NumClustersStartDeleteOldest))
	log.Logfln("  DurationForClusterDeletion: " + o.DurationForClusterDeletion)

	shootClusterManager, err := pkg.NewShootClusterManager(log, o.GardenClusterKubeconfigPath, o.Namespace,
		o.AuthDirectoryPath, o.MaxNumOfClusters, o.NumClustersStartDeleteOldest, o.DurationForClusterDeletion)

	if err != nil {
		return err
	}

	if err := shootClusterManager.CreateShootCluster(ctx); err != nil {
		return err
	}

	return nil
}
