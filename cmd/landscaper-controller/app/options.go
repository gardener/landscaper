// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"
	"io/ioutil"
	"strings"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/version"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerinstall "github.com/gardener/landscaper/apis/deployer/container/install"
	helminstall "github.com/gardener/landscaper/apis/deployer/helm/install"
	manifestinstall "github.com/gardener/landscaper/apis/deployer/manifest/install"
	mockinstall "github.com/gardener/landscaper/apis/deployer/mock/install"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/constants"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string

	config   *config.LandscaperConfiguration
	deployer deployerOptions
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.deployer.deployers, "deployers", "",
		`Specify additional deployers that should be enabled.
Controllers are specified as a comma separated list of controller names.
Available deployers are mock,helm,container.`)
	fs.StringVar(&o.deployer.Version, "deployer-version", version.Get().String(),
		"set the version for the automatically deployed deployers.")
	fs.StringVar(&o.deployer.deployersConfigPath, "deployers-config", "", "Specify the path to the deployers-configuration file")
	logger.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	log, err := logger.New(nil)
	if err != nil {
		return err
	}
	o.log = log.WithName("setup")
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	o.config, err = o.parseConfigurationFile()
	if err != nil {
		return err
	}
	if err := o.deployer.Complete(); err != nil {
		return err
	}

	err = o.validate() // validate options
	if err != nil {
		return err
	}

	return nil
}

func (o *options) parseConfigurationFile() (*config.LandscaperConfiguration, error) {
	decoder := serializer.NewCodecFactory(api.ConfigScheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		cfg := &config.LandscaperConfiguration{}
		api.ConfigScheme.Default(cfg)
		return cfg, nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return nil, err
	}

	cfg := &config.LandscaperConfiguration{}
	if _, _, err := decoder.Decode(data, nil, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validates the options
func (o *options) validate() error {
	return nil
}

//////////////////////
// Deployer Options //
//////////////////////

type deployerOptions struct {
	deployers           string
	deployersConfigPath string

	EnabledDeployers []string
	Version          string
	DeployersConfig  DeployersConfiguration
}

func (o *deployerOptions) GetDeployerConfiguration(name string, config runtime.Object) error {
	if o.DeployersConfig.Deployers == nil {
		return nil
	}
	data, ok := o.DeployersConfig.Deployers[name]
	if !ok || data.Raw == nil {
		return nil
	}
	deployerScheme := runtime.NewScheme()
	helminstall.Install(deployerScheme)
	manifestinstall.Install(deployerScheme)
	containerinstall.Install(deployerScheme)
	mockinstall.Install(deployerScheme)

	if _, _, err := serializer.NewCodecFactory(deployerScheme).UniversalDecoder().Decode(data.Raw, nil, config); err != nil {
		return err
	}
	return nil
}

func (o *deployerOptions) Complete() error {
	if len(o.deployers) != 0 {
		o.EnabledDeployers = strings.Split(o.deployers, ",")
	}

	if err := o.parseDeployersConfigurationFile(); err != nil {
		return err
	}

	return nil
}

func (o *deployerOptions) parseDeployersConfigurationFile() error {
	if len(o.deployersConfigPath) == 0 {
		return nil
	}
	data, err := ioutil.ReadFile(o.deployersConfigPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, &o.DeployersConfig)
}

func addDefaultTargetSelector(selectors []lsv1alpha1.TargetSelector) []lsv1alpha1.TargetSelector {
	if selectors == nil {
		selectors = make([]lsv1alpha1.TargetSelector, 0)
	}
	selectors = append(selectors, lsv1alpha1.TargetSelector{
		Annotations: []lsv1alpha1.Requirement{
			{
				Key:      constants.NotUseDefaultDeployerAnnotation,
				Operator: selection.DoesNotExist,
			},
		},
	})

	return selectors
}
