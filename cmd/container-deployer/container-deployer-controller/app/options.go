// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/logger"
)

type options struct {
	log                  logr.Logger
	configPath           string
	landscaperKubeconfig string

	config                      *containerv1alpha1.Configuration
	landscaperClusterRestConfig *rest.Config
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.landscaperKubeconfig, "landscaper-kubeconfig", "", "Specify the path to the landscaper kubeconfig cluster")
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

	if len(o.landscaperKubeconfig) != 0 {
		data, err := ioutil.ReadFile(o.landscaperKubeconfig)
		if err != nil {
			return fmt.Errorf("unable to read landscaper kubeconfig from %s: %w", o.landscaperKubeconfig, err)
		}
		client, err := clientcmd.NewClientConfigFromBytes(data)
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster client from %s: %w", o.landscaperKubeconfig, err)
		}
		o.landscaperClusterRestConfig, err = client.ClientConfig()
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster rest client from %s: %w", o.landscaperKubeconfig, err)
		}
	}

	return nil
}

func (o *options) parseConfigurationFile() (*containerv1alpha1.Configuration, error) {
	decoder := serializer.NewCodecFactory(container.Scheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		return &containerv1alpha1.Configuration{}, nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return nil, err
	}

	cfg := &containerv1alpha1.Configuration{}
	if _, _, err := decoder.Decode(data, nil, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
