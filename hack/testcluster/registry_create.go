// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/hack/testcluster/pkg"
)

func NewCreateRegistryCommand(ctx context.Context) *cobra.Command {
	opts := &CreateRegistryOptions{}
	cmd := &cobra.Command{
		Use:          "create",
		Short:        "creates a new test registry running in a pod",
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

// CreateRegistryOptions defines all options that are needed for create registry command.
type CreateRegistryOptions struct {
	CommonOptions
	// ExportRegistryCreds is the path to the file where the credentials for the registry should be written to.
	// The credentials are output as valid docker auth config.
	ExportRegistryCreds string
	// Password is the password that should be used for the registry basic auth.
	// Will be generated if not provided
	Password string

	// DNSFormat is the type of the output address for the registry
	// Can be either internal or external
	// internal will result in a hostname routable only via in-cluster-DNS, while external will result in a publicly routable IP or hostname.
	DNSFormat string

	RunOnShoot bool
}

// AddFlags adds flags for the options to a flagset
func (o *CreateRegistryOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}
	o.CommonOptions.AddFlags(fs)
	fs.StringVar(&o.Password, "registry-password", "", "set the registry password")
	fs.StringVar(&o.ExportRegistryCreds, "registry-auth", "", "path where the docker auth config is written to")
	fs.StringVar(&o.DNSFormat, "dns-format", "internal", "determines the type of address which is used for the registry service. Can be 'internal' (uses in-cluster-DNS) or 'external' (is publicly reachable).")
	fs.BoolVar(&o.RunOnShoot, "ls-run-on-shoot", false, "runs on a shoot and not a k3s cluster")

}

func (o *CreateRegistryOptions) Complete() error {
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
	if len(o.Password) == 0 {
		o.Password = pkg.RandString(10)
	}
	return o.CommonOptions.Complete()
}

func (o *CreateRegistryOptions) Validate() error {
	return o.CommonOptions.Validate()
}

func (o *CreateRegistryOptions) Run(ctx context.Context) error {
	logger := utils.NewLogger().WithTimestamp()
	return pkg.CreateRegistry(ctx,
		logger,
		o.kubeClient,
		o.restConfig,
		o.Namespace,
		o.ID,
		o.StateFile,
		o.Password,
		o.DNSFormat,
		o.ExportRegistryCreds,
		o.Timeout,
		o.RunOnShoot)
}
