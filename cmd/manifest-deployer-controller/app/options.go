// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	goflag "flag"
	"io/ioutil"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"

	manifestv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/manifest/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string

	config *manifestv1alpha1.Configuration
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

func (o *options) parseConfigurationFile() (*manifestv1alpha1.Configuration, error) {
	decoder := serializer.NewCodecFactory(manifest.ManifestScheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		return &manifestv1alpha1.Configuration{}, nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return nil, err
	}

	cfg := &manifestv1alpha1.Configuration{}
	if _, _, err := decoder.Decode(data, nil, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
