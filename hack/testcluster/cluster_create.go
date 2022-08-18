// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/hack/testcluster/pkg"
	"github.com/gardener/landscaper/test/utils"
)

func NewCreateClusterCommand(ctx context.Context) *cobra.Command {
	opts := &CreateClusterOptions{}
	cmd := &cobra.Command{
		Use:          "create",
		Short:        "creates a new test cluster running in a pod",
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

// CreateClusterOptions defines all options that are needed for create cluster commmand.
type CreateClusterOptions struct {
	CommonOptions
	// ExportKubeconfigPath is the path to the test cluster kubeconfig
	ExportKubeconfigPath string
	// KubernetesVersion defines the kubernetes version of the cluster.
	KubernetesVersion string
}

// AddFlags adds flags for the options to a flagset
func (o *CreateClusterOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}
	o.CommonOptions.AddFlags(fs)
	fs.StringVar(&o.ExportKubeconfigPath, "export", "", "path where the target kubeconfig should be written to")
	fs.StringVar(&o.KubernetesVersion, "kubernetes-version", pkg.DefaultK8sVersion, "specify the kubernetes version of the cluster")
}

func (o *CreateClusterOptions) Complete() error {
	// generate id if none is defined
	if len(o.ID) == 0 {
		uid, err := uuid.NewUUID()
		if err != nil {
			return fmt.Errorf("unable to generate uuid: %w", err)
		}
		o.ID = base64.StdEncoding.EncodeToString([]byte(uid.String()))
	}
	if err := o.Validate(); err != nil {
		return err
	}
	return o.CommonOptions.Complete()
}

func (o *CreateClusterOptions) Validate() error {
	return o.CommonOptions.Validate()
}

func (o *CreateClusterOptions) Run(ctx context.Context) error {
	logger := utils.NewLogger().WithTimestamp()
	return pkg.CreateCluster(ctx, logger, pkg.CreateClusterArgs{
		KubeClient:           o.kubeClient,
		RestConfig:           o.restConfig,
		Namespace:            o.Namespace,
		ID:                   o.ID,
		StateFile:            o.StateFile,
		ExportKubeconfigPath: o.ExportKubeconfigPath,
		Timeout:              o.Timeout,
		KubernetesVersion:    o.KubernetesVersion,
	})
}
