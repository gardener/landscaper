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
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string
	deployers  string

	config           *config.LandscaperConfiguration
	enabledDeployers []string
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.deployers, "deployers", "",
		`Specify additional deployers that should be enabled.
Controllers are specified as a comma separated list of controller names.
Available deployers are mock,helm,container.`)
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

	if len(o.deployers) != 0 {
		o.enabledDeployers = strings.Split(o.deployers, ",")
	}

	return nil
}

func (o *options) parseConfigurationFile() (*config.LandscaperConfiguration, error) {
	decoder := serializer.NewCodecFactory(kubernetes.ConfigScheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		return &config.LandscaperConfiguration{}, nil
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
