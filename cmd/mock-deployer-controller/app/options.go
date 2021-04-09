// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"
	"io/ioutil"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"

	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	mockctrl "github.com/gardener/landscaper/pkg/deployer/mock"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string

	config *mockv1alpha1.Configuration
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

	return o.parseConfig()
}

func (o *options) parseConfig() error {
	if len(o.configPath) == 0 {
		o.config = &mockv1alpha1.Configuration{}
		mockctrl.MockScheme.Default(o.config)
		return nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return err
	}

	o.config = &mockv1alpha1.Configuration{}
	if _, _, err := mockctrl.Decoder.Decode(data, nil, o.config); err != nil {
		return err
	}
	return nil
}
