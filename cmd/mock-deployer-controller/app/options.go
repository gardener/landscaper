// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	flag "github.com/spf13/pflag"

	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	deployercmd "github.com/gardener/landscaper/pkg/deployer/lib/cmd"
	mockctrl "github.com/gardener/landscaper/pkg/deployer/mock"
)

type options struct {
	DeployerOptions *deployercmd.DefaultOptions
	Config          mockv1alpha1.Configuration
}

func NewOptions() *options {
	return &options{
		DeployerOptions: deployercmd.NewDefaultOptions(mockctrl.MockScheme),
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
