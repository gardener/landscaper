// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"
	"io/ioutil"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/landscaper/apis/deployer/manifest"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string

	config *manifest.Configuration
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
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

	return nil
}

func (o *options) parseConfigurationFile() (*manifest.Configuration, error) {
	decoder := serializer.NewCodecFactory(manifestctlr.ManifestScheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		return &manifest.Configuration{}, nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return nil, err
	}

	cfg := &manifest.Configuration{}
	if _, _, err := decoder.Decode(data, nil, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
