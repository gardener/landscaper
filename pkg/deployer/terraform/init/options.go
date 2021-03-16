// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"

	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/terraform/terraformer"
)

// Options describe all options for the terraform init container.
type Options struct {
	ConfigurationFilePath  string
	RegistrySecretBasePath string
	SharedDirPath          string
	ProvidersDir           string

	Configuration *terraformv1alpha1.ProviderConfiguration
}

// Complete reads necessary options from the expected sources.
func (o *Options) Complete(ctx context.Context) {
	o.ConfigurationFilePath = os.Getenv(terraformer.DeployItemConfigurationPathName)
	o.RegistrySecretBasePath = os.Getenv(terraformer.RegistrySecretBasePathName)
	o.SharedDirPath = os.Getenv(terraformer.TerraformSharedDirEnvVarName)
	o.ProvidersDir = os.Getenv(terraformer.TerraformProvidersDirEnvVarName)
}

// Validate validates the options data.
func (o *Options) Validate() error {
	var err *multierror.Error
	if len(o.ConfigurationFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", terraformer.DeployItemConfigurationPathName))
	}
	if len(o.SharedDirPath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", terraformer.TerraformSharedDirEnvVarName))
	}
	if len(o.ProvidersDir) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", terraformer.TerraformProvidersDirEnvVarName))
	}
	return err.ErrorOrNil()
}
