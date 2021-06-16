// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"io/ioutil"
	"strings"

	flag "github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/version"
)

type Options struct {
	Deployers           string
	DeployersConfigPath string

	EnabledDeployers []string
	Version          string
	DeployersConfig  DeployersConfiguration
}

// AddFlags adds the flags for the deployer management.
func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.Deployers, "deployers", "",
		`Specify additional Deployers that should be enabled.
Controllers are specified as a comma separated list of controller names.
Available Deployers are mock,helm,container.`)
	fs.StringVar(&o.Version, "deployer-version", version.Get().String(),
		"set the version for the automatically deployed Deployers.")
	fs.StringVar(&o.DeployersConfigPath, "deployers-config", "", "Specify the path to the deployers-configuration file")
}

// Complete parses the provided deployer management configurations.
func (o *Options) Complete() error {
	if len(o.Deployers) != 0 {
		o.EnabledDeployers = strings.Split(o.Deployers, ",")
	}

	if err := o.parseDeployersConfigurationFile(); err != nil {
		return err
	}

	return nil
}

// GetDeployerConfigForDeployer returns the optional configuration for a deployer.
// If not config is defined a NoConfigError is returned
func (o *Options) GetDeployerConfigForDeployer(deployerName string) (DeployerConfiguration, error) {
	data, ok := o.DeployersConfig.Deployers[deployerName]
	if !ok {
		return DeployerConfiguration{}, NoConfigError
	}
	return data, nil
}

func (o *Options) parseDeployersConfigurationFile() error {
	if len(o.DeployersConfigPath) == 0 {
		return nil
	}
	data, err := ioutil.ReadFile(o.DeployersConfigPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, &o.DeployersConfig)
}
