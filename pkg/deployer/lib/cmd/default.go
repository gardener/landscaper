// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"time"

	"k8s.io/utils/pointer"

	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	lsinstall "github.com/gardener/landscaper/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/lib"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// DefaultOptions defines all default deployer options.
type DefaultOptions struct {
	LsUncachedClient   client.Client
	LsCachedClient     client.Client
	HostUncachedClient client.Client
	HostCachedClient   client.Client

	configPath   string
	LsKubeconfig string

	Log     logging.Logger
	LsMgr   manager.Manager
	HostMgr manager.Manager

	decoder runtime.Decoder

	FinishedObjectCache *lsutils.FinishedObjectCache
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
	logging.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *DefaultOptions) Complete() error {
	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	log = log.WithName("deployer")
	o.Log = log
	ctrl.SetLogger(log.Logr())
	ctx := logging.NewContext(context.Background(), o.Log)

	hostAndResourceClusterDifferent := len(o.LsKubeconfig) != 0

	burst, qps := lsutils.GetHostClientRequestRestrictions(log, hostAndResourceClusterDifferent)

	opts := manager.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0", // disable the metrics serving by default
		SyncPeriod:         pointer.Duration(time.Hour * 24 * 1000),
	}

	hostRestConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get host kubeconfig: %w", err)
	}
	hostRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(log, hostRestConfig, burst, qps)

	o.HostMgr, err = ctrl.NewManager(hostRestConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup host manager")
	}
	o.LsMgr = o.HostMgr

	if hostAndResourceClusterDifferent {
		data, err := os.ReadFile(o.LsKubeconfig)
		if err != nil {
			return fmt.Errorf("unable to read landscaper kubeconfig from %s: %w", o.LsKubeconfig, err)
		}

		lsRestConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster rest client: %w", err)
		}
		burst, qps = lsutils.GetResourceClientRequestRestrictions(log)
		lsRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(log, lsRestConfig, burst, qps)

		o.LsMgr, err = ctrl.NewManager(lsRestConfig, opts)
		if err != nil {
			return fmt.Errorf("unable to setup ls manager")
		}
	}

	lsinstall.Install(o.LsMgr.GetScheme())

	o.LsUncachedClient, o.LsCachedClient, o.HostUncachedClient, o.HostCachedClient, err = lsutils.ClientsFromManagers(o.LsMgr, o.HostMgr)
	if err != nil {
		return err
	}

	if err := o.prepareFinishedObjectCache(ctx); err != nil {
		return err
	}

	return nil
}

func (o *DefaultOptions) prepareFinishedObjectCache(ctx context.Context) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	o.FinishedObjectCache = lsutils.NewFinishedObjectCache()
	namespaces := &v1.NamespaceList{}
	if err := read_write_layer.ListNamespaces(ctx, o.LsUncachedClient, namespaces, read_write_layer.R000093); err != nil {
		return err
	}

	perfTotal := lsutils.StartPerformanceMeasurement(&log, "prepare finished object for dis")
	defer perfTotal.Stop()

	for _, namespace := range namespaces.Items {
		perf := lsutils.StartPerformanceMeasurement(&log, "prepare finished object cache for dis: fetch from namespace "+namespace.Name)

		diList := &lsv1alpha1.DeployItemList{}
		if err := read_write_layer.ListDeployItems(ctx, o.LsUncachedClient, diList, read_write_layer.R000094,
			client.InNamespace(namespace.Name)); err != nil {
			return err
		}

		perf.Stop()

		perf = lsutils.StartPerformanceMeasurement(&log, "prepare finished object cache for dis: add for namespace "+namespace.Name)

		for diIndex := range diList.Items {
			di := &diList.Items[diIndex]
			if lib.IsDeployItemFinished(di) {
				o.FinishedObjectCache.Add(&di.ObjectMeta)
			}
		}

		perf.Stop()
	}

	return nil
}

// StartManagers starts the host and landscaper managers.
func (o *DefaultOptions) StartManagers(ctx context.Context, deployerJobs ...DeployerJob) error {
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

	for i := range deployerJobs {
		nextJob := deployerJobs[i]
		eg.Go(func() error {
			if err := nextJob.StartDeployerJob(ctx); err != nil {
				return fmt.Errorf("error while running deployerJob: %w", err)
			}
			return nil
		})
	}
	return eg.Wait()
}

// GetConfig reads and parses the configured configuration file.
func (o *DefaultOptions) GetConfig(obj runtime.Object) error {
	if len(o.configPath) == 0 {
		return nil
	}
	data, err := os.ReadFile(o.configPath)
	if err != nil {
		return fmt.Errorf("uable to read config from %q: %w", o.configPath, err)
	}

	if _, _, err := o.decoder.Decode(data, nil, obj); err != nil {
		return err
	}

	if o.Log.Enabled(logging.INFO) {
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
