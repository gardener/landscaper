// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	goflag "flag"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/controller-utils/pkg/logger"
	"github.com/gardener/landscaper/pkg/api"

	lsinstall "github.com/gardener/landscaper/apis/core/install"
)

// DefaultOptions defines all default deployer options.
type DefaultOptions struct {
	configPath   string
	LsKubeconfig string

	Log     logr.Logger
	LsMgr   manager.Manager
	HostMgr manager.Manager

	decoder runtime.Decoder
}

// NewDefaultOptions creates new default options for a deployer.
func NewDefaultOptions(deployerScheme *runtime.Scheme) *DefaultOptions {
	return &DefaultOptions{
		decoder: api.NewDecoder(deployerScheme),
	}
}

func (o *DefaultOptions) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.LsKubeconfig, "landscaper-kubeconfig", "", "Specify the path to the landscaper kubeconfig cluster")
	logger.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *DefaultOptions) Complete() error {
	log, err := logger.New(nil)
	if err != nil {
		return err
	}
	o.Log = log.WithName("setup")
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	opts := manager.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0", // disable the metrics serving by default
	}

	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get host kubeconfig: %w", err)
	}
	o.HostMgr, err = ctrl.NewManager(restConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager")
	}
	o.LsMgr = o.HostMgr

	if len(o.LsKubeconfig) != 0 {
		data, err := ioutil.ReadFile(o.LsKubeconfig)
		if err != nil {
			return fmt.Errorf("unable to read landscaper kubeconfig from %s: %w", o.LsKubeconfig, err)
		}
		client, err := clientcmd.NewClientConfigFromBytes(data)
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster client from %s: %w", o.LsKubeconfig, err)
		}
		restConfig, err := client.ClientConfig()
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster rest client from %s: %w", o.LsKubeconfig, err)
		}

		o.LsMgr, err = ctrl.NewManager(restConfig, opts)
		if err != nil {
			return fmt.Errorf("unable to setup manager")
		}
	}

	lsinstall.Install(o.LsMgr.GetScheme())

	return nil
}

// StartManagers starts the host and landscaper managers.
func (o *DefaultOptions) StartManagers(ctx context.Context) error {
	o.Log.Info("Starting the controllers")
	eg, ctx := errgroup.WithContext(ctx)

	if o.LsMgr != o.HostMgr {
		eg.Go(func() error {
			if err := o.HostMgr.Start(ctx); err != nil {
				return fmt.Errorf("error while running host manager: %w", err)
			}
			return nil
		})
		o.Log.Info("Waiting for host cluster cache to sync")
		if !o.HostMgr.GetCache().WaitForCacheSync(ctx) {
			return errors.New("unable to sync host cluster cache")
		}
		o.Log.Info("Cache of host cluster successfully synced")
	}
	eg.Go(func() error {
		if err := o.LsMgr.Start(ctx); err != nil {
			return fmt.Errorf("error while running landscaper manager: %w", err)
		}
		return nil
	})
	return eg.Wait()
}

// GetConfig reads and parses the configured configuration file.
func (o *DefaultOptions) GetConfig(obj runtime.Object) error {
	if len(o.configPath) == 0 {
		return nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return fmt.Errorf("uable to read config from %q: %w", o.configPath, err)
	}

	if _, _, err := o.decoder.Decode(data, nil, obj); err != nil {
		return err
	}

	if o.Log.V(2).Enabled() {
		// print configuration if enabled
		configBytes, err := yaml.Marshal(obj)
		if err != nil {
			o.Log.Error(err, "unable to marshal configuration")
		} else {
			fmt.Println(string(configBytes))
		}
	}
	return nil
}
