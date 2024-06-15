// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	contextctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/context"
	deployitemctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	executionactrl "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/healthcheck"
	installationsctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/targetsync"
	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
	"github.com/gardener/landscaper/pkg/metrics"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"
	"github.com/gardener/landscaper/pkg/utils/monitoring"
	"github.com/gardener/landscaper/pkg/version"
)

// NewLandscaperControllerCommand creates a new landscaper command that runs the landscaper controller.
func NewLandscaperControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "landscaper-controller",
		Short: "Landscaper controller manages the orchestration of components",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(ctx); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(ctx); err != nil {
				options.Log.Error(err, "unable to run landscaper controller")
				os.Exit(1)
			}
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) run(ctx context.Context) error {
	setupLogger := o.Log.WithName("setup")
	setupLogger.Info("Starting Landscaper Controller", lc.KeyVersion, version.Get().String())

	configBytes, err := yaml.Marshal(o.Config)
	if err != nil {
		return fmt.Errorf("unable to marshal Landscaper config: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stderr, string(configBytes))

	hostAndResourceClusterDifferent := len(o.landscaperKubeconfigPath) > 0

	burst, qps := lsutils.GetHostClientRequestRestrictions(setupLogger, hostAndResourceClusterDifferent)

	opts := manager.Options{
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
		Cache:          cache.Options{SyncPeriod: ptr.To[time.Duration](time.Hour * 24 * 1000)},
	}

	//TODO: investigate whether this is used with an uncached client
	if o.Config.Controllers.SyncPeriod != nil {
		opts.Cache.SyncPeriod = &o.Config.Controllers.SyncPeriod.Duration
	}

	if o.Config.Metrics != nil {
		opts.Metrics.BindAddress = fmt.Sprintf(":%d", o.Config.Metrics.Port)
	}

	hostRestConfig := ctrl.GetConfigOrDie()
	hostRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(setupLogger, hostRestConfig, burst, qps)

	hostMgr, err := ctrl.NewManager(hostRestConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	lsMgr := hostMgr
	if hostAndResourceClusterDifferent {
		data, err := os.ReadFile(o.landscaperKubeconfigPath)
		if err != nil {
			return fmt.Errorf("unable to read landscaper kubeconfig from %s: %w", o.landscaperKubeconfigPath, err)
		}

		lsRestConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster rest client: %w", err)
		}
		burst, qps = lsutils.GetResourceClientRequestRestrictions(setupLogger)
		lsRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(setupLogger, lsRestConfig, burst, qps)

		lsMgr, err = ctrl.NewManager(lsRestConfig, opts)
		if err != nil {
			return fmt.Errorf("unable to setup landscaper cluster manager from %s: %w", o.landscaperKubeconfigPath, err)
		}
	}

	metrics.RegisterMetrics(controllerruntimeMetrics.Registry)
	ctrlLogger := o.Log.WithName("controllers")

	if err := o.ensureCRDs(ctx, lsMgr); err != nil {
		return err
	}
	if lsMgr != hostMgr {
		if err := o.ensureCRDs(ctx, hostMgr); err != nil {
			return err
		}
	}
	install.Install(lsMgr.GetScheme())

	lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient, err := lsutils.ClientsFromManagers(lsMgr, hostMgr)
	if err != nil {
		return err
	}

	if os.Getenv("LANDSCAPER_MODE") == "central-landscaper" {
		return o.startCentralLandscaper(ctx, lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
			lsMgr, hostMgr, ctrlLogger, setupLogger)
	} else {
		return o.startMainController(ctx, lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
			lsMgr, hostMgr, ctrlLogger, setupLogger)
	}
}

func (o *Options) startMainController(ctx context.Context,
	lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	lsMgr, hostMgr manager.Manager, ctrlLogger, setupLogger logging.Logger) error {

	if err := installationsctrl.AddControllerToManager(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		ctrlLogger, lsMgr, o.Config, "installations"); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactrl.AddControllerToManager(ctx, lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		ctrlLogger, lsMgr, hostMgr, o.Config); err != nil {
		return fmt.Errorf("unable to setup execution controller: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	setupLogger.Info("starting the controllers")
	if lsMgr != hostMgr {
		eg.Go(func() error {
			if err := hostMgr.Start(ctx); err != nil {
				return fmt.Errorf("error while running host manager: %w", err)
			}
			return nil
		})
		setupLogger.Info("Waiting for host cluster cache to sync")
		if !hostMgr.GetCache().WaitForCacheSync(ctx) {
			return fmt.Errorf("unable to sync host cluster cache")
		}
		setupLogger.Info("Cache of host cluster successfully synced")
	}
	eg.Go(func() error {
		if err := lsMgr.Start(ctx); err != nil {
			return fmt.Errorf("error while running landscaper manager: %w", err)
		}
		return nil
	})
	return eg.Wait()
}

func (o *Options) startCentralLandscaper(ctx context.Context,
	lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	lsMgr, hostMgr manager.Manager, ctrlLogger, setupLogger logging.Logger) error {

	if err := contextctrl.AddControllerToManager(lsUncachedClient, lsCachedClient, ctrlLogger, lsMgr, o.Config); err != nil {
		return fmt.Errorf("unable to setup context controller: %w", err)
	}

	if err := deployitemctrl.AddControllerToManager(lsUncachedClient, lsCachedClient,
		ctrlLogger,
		lsMgr,
		o.Config.Controllers.DeployItems,
		o.Config.DeployItemTimeouts.Pickup); err != nil {
		return fmt.Errorf("unable to setup deployitem controller: %w", err)
	}

	if err := targetsync.AddControllerToManagerForTargetSyncs(lsUncachedClient, lsCachedClient, ctrlLogger, lsMgr); err != nil {
		return fmt.Errorf("unable to register target sync controller: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		healthChecker := healthcheck.NewHealthChecker(o.Config.LsDeployments, hostUncachedClient)
		if err := healthChecker.StartPeriodicalHealthCheck(ctx, ctrlLogger); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		lockCleaner := lock.NewLockCleaner(lsUncachedClient)
		lockCleaner.StartPeriodicalSyncObjectCleanup(ctx, ctrlLogger)
		return nil
	})

	eg.Go(func() error {
		monitor := monitoring.NewMonitor(lsutils.GetCurrentPodNamespace(), hostUncachedClient)
		monitor.StartMonitoring(ctx, ctrlLogger)
		return nil
	})

	setupLogger.Info("starting the controllers")
	if lsMgr != hostMgr {
		eg.Go(func() error {
			if err := hostMgr.Start(ctx); err != nil {
				return fmt.Errorf("error while running host manager: %w", err)
			}
			return nil
		})
		setupLogger.Info("Waiting for host cluster cache to sync")
		if !hostMgr.GetCache().WaitForCacheSync(ctx) {
			return fmt.Errorf("unable to sync host cluster cache")
		}
		setupLogger.Info("Cache of host cluster successfully synced")
	}
	eg.Go(func() error {
		if err := lsMgr.Start(ctx); err != nil {
			return fmt.Errorf("error while running landscaper manager: %w", err)
		}
		return nil
	})
	return eg.Wait()
}

func (o *Options) ensureCRDs(ctx context.Context, mgr manager.Manager) error {
	ctx = logging.NewContext(ctx, logging.Wrap(ctrl.Log.WithName("crdManager")))
	crdmgr, err := crdmanager.NewCrdManager(mgr, o.Config)
	if err != nil {
		return fmt.Errorf("unable to setup CRD manager: %w", err)
	}

	if err := crdmgr.EnsureCRDs(ctx); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	return nil
}
