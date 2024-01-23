// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"time"

	"k8s.io/utils/pointer"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	lsinstall "github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	lsutils "github.com/gardener/landscaper/pkg/utils"
)

type options struct {
	log                      logging.Logger
	configPath               string
	landscaperKubeconfigPath string

	config  config.AgentConfiguration
	LsMgr   manager.Manager
	HostMgr manager.Manager
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.landscaperKubeconfigPath, "landscaper-kubeconfig", "", "Specify the path to the landscaper kubeconfig cluster")
	logging.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	o.log = log
	ctrl.SetLogger(log.Logr())

	o.config, err = o.parseConfigurationFile()
	if err != nil {
		return err
	}

	err = o.validate() // validate options
	if err != nil {
		return err
	}

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
		SyncPeriod:         pointer.Duration(time.Hour * 24 * 1000),
	}

	hostRestConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get host kubeconfig: %w", err)
	}
	hostRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(log, hostRestConfig, lsutils.LsHostClientBurstDefault, lsutils.LsHostClientQpsDefault)

	o.HostMgr, err = ctrl.NewManager(hostRestConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup host manager")
	}

	data, err := os.ReadFile(o.landscaperKubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to read landscaper kubeconfig from %s: %w", o.landscaperKubeconfigPath, err)
	}

	lsRestConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return fmt.Errorf("unable to build landscaper cluster rest client for agent: %w", err)
	}
	lsRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(log, lsRestConfig, lsutils.LsResourceClientBurstDefault, lsutils.LsResourceClientQpsDefault)

	o.LsMgr, err = ctrl.NewManager(lsRestConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager")
	}
	lsinstall.Install(o.LsMgr.GetScheme())

	return nil
}

func (o *options) parseConfigurationFile() (config.AgentConfiguration, error) {
	decoder := serializer.NewCodecFactory(api.ConfigScheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		return config.AgentConfiguration{}, nil
	}
	data, err := os.ReadFile(o.configPath)
	if err != nil {
		return config.AgentConfiguration{}, err
	}

	cfg := config.AgentConfiguration{}
	if _, _, err := decoder.Decode(data, nil, &cfg); err != nil {
		return config.AgentConfiguration{}, err
	}

	return cfg, nil
}

// validates the options
func (o *options) validate() error {
	if len(o.landscaperKubeconfigPath) == 0 {
		return errors.New("the landscaper kubeconfig has to be provided")
	}
	return nil
}
