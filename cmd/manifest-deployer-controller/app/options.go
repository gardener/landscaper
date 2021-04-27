// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	flag "github.com/spf13/pflag"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	deployercmd "github.com/gardener/landscaper/pkg/deployer/lib/cmd"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
)

type options struct {
	DeployerOptions *deployercmd.DefaultOptions
	Config          manifestv1alpha2.Configuration
}

func NewOptions() *options {
	return &options{
		DeployerOptions: deployercmd.NewDefaultOptions(manifest.Scheme),
	}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	o.DeployerOptions.AddFlags(fs)
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	if err := o.DeployerOptions.Complete(); err != nil {
		return err
	}
	if err := o.DeployerOptions.GetConfig(&o.Config); err != nil {
		return err
	}
	return nil
}
