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

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/kubernetes"
	blueprintregistrymanager "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints/manager"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string

	config           *config.LandscaperConfiguration
	registry         blueprintregistrymanager.Interface
	cRegistryManager *componentsregistry.Manager
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

//func (o *options) setupRegistry() error {
//	r, err := blueprintregistrymanager.NewWithConfig(o.log, &o.config.Registry)
//	if err != nil {
//		return err
//	}
//	o.registry = r
//	return errors.New("no registry defined")
//}

//func (o *options) setupComponentsRegistryManager() error {
//	clients := make([]componentsregistry.TypedRegistry, 0)
//	if o.config.ComponentDescriptorRepository.Local != nil {
//		localClient, err := componentsregistry.NewLocalClient(o.log, o.config.ComponentDescriptorRepository.Local.Paths...)
//		if err != nil {
//			return err
//		}
//		clients = append(clients, localClient)
//	}
//	if o.config.ComponentDescriptorRepository.OCI != nil {
//		localClient, err := componentsregistry.NewOCIRegistry(o.log, o.config.ComponentDescriptorRepository.OCI)
//		if err != nil {
//			return err
//		}
//		clients = append(clients, localClient)
//	}
//	o.cRegistryManager = componentsregistry.New(nil, clients...)
//	return nil
//}

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
