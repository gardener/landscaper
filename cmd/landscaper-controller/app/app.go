// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/agent"
	deployers "github.com/gardener/landscaper/pkg/deployermanagement/controller"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
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
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
		NewClient:          lsutils.NewUncachedClient(burst, qps),
	}

	//TODO: investigate whether this is used with an uncached client
	if o.Config.Controllers.SyncPeriod != nil {
		opts.Cache.SyncPeriod = &o.Config.Controllers.SyncPeriod.Duration
	}

	if o.Config.Metrics != nil {
		opts.MetricsBindAddress = fmt.Sprintf(":%d", o.Config.Metrics.Port)
	}

	hostMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	lsMgr := hostMgr
	if hostAndResourceClusterDifferent {
		data, err := os.ReadFile(o.landscaperKubeconfigPath)
		if err != nil {
			return fmt.Errorf("unable to read landscaper kubeconfig from %s: %w", o.landscaperKubeconfigPath, err)
		}
		client, err := clientcmd.NewClientConfigFromBytes(data)
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster client from %s: %w", o.landscaperKubeconfigPath, err)
		}
		lsConfig, err := client.ClientConfig()
		if err != nil {
			return fmt.Errorf("unable to build landscaper cluster rest client from %s: %w", o.landscaperKubeconfigPath, err)
		}

		opts.MetricsBindAddress = "0"
		burst, qps = lsutils.GetResourceClientRequestRestrictions(setupLogger)
		opts.NewClient = lsutils.NewUncachedClient(burst, qps)

		lsMgr, err = ctrl.NewManager(lsConfig, opts)
		if err != nil {
			return fmt.Errorf("unable to setup landscaper cluster manager from %s: %w", o.landscaperKubeconfigPath, err)
		}
	}

	metrics.RegisterMetrics(controllerruntimeMetrics.Registry)
	ctrlLogger := o.Log.WithName("controllers")

	if os.Getenv("LANDSCAPER_MODE") == "central-landscaper" {
		return o.startCentralLandscaper(ctx, lsMgr, hostMgr, ctrlLogger, setupLogger)
	} else {
		return o.startMainController(ctx, lsMgr, hostMgr, ctrlLogger, setupLogger)
	}

}

func (o *Options) startMainController(ctx context.Context, lsMgr, hostMgr manager.Manager,
	ctrlLogger, setupLogger logging.Logger) error {
	install.Install(lsMgr.GetScheme())

	store, err := blueprints.NewStore(o.Log.WithName("blueprintStore"), osfs.New(), o.Config.BlueprintStore)
	if err != nil {
		return fmt.Errorf("unable to setup blueprint store: %w", err)
	}
	blueprints.SetStore(store)

	if err := installationsctrl.AddControllerToManager(ctrlLogger, lsMgr, hostMgr, o.Config, "installations"); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactrl.AddControllerToManager(ctrlLogger, lsMgr, hostMgr, o.Config); err != nil {
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

func (o *Options) startCentralLandscaper(ctx context.Context, lsMgr, hostMgr manager.Manager,
	ctrlLogger, setupLogger logging.Logger) error {

	if err := o.ensureCRDs(ctx, lsMgr); err != nil {
		return err
	}

	if lsMgr != hostMgr {
		if err := o.ensureCRDs(ctx, hostMgr); err != nil {
			return err
		}
	}

	install.Install(lsMgr.GetScheme())

	if err := contextctrl.AddControllerToManager(ctrlLogger, lsMgr, o.Config); err != nil {
		return fmt.Errorf("unable to setup context controller: %w", err)
	}

	if !o.Config.DeployerManagement.Disable {
		if err := deployers.AddControllersToManager(ctrlLogger, lsMgr, o.Config); err != nil {
			return fmt.Errorf("unable to setup deployer controllers: %w", err)
		}
		if !o.Config.DeployerManagement.Agent.Disable {
			agentConfig := o.Config.DeployerManagement.Agent.AgentConfiguration
			// add default selector and in addition reconcile all target that do not have a a environment definition
			agentConfig.TargetSelectors = append(agent.DefaultTargetSelector(agentConfig.Name),
				lsv1alpha1.TargetSelector{
					Annotations: []lsv1alpha1.Requirement{
						{
							Key:      lsv1alpha1.DeployerEnvironmentTargetAnnotationName,
							Operator: selection.DoesNotExist,
						},
					},
				},
			)
			if err := agent.AddToManager(ctx, o.Log, lsMgr, hostMgr, agentConfig, "landscaper-helm"); err != nil {
				return fmt.Errorf("unable to setup default agent: %w", err)
			}
		}
		if err := o.DeployInternalDeployers(ctx, lsMgr); err != nil {
			return err
		}

		if o.Config.LsDeployments.AdditionalDeployments == nil {
			o.Config.LsDeployments.AdditionalDeployments = &config.AdditionalDeployments{
				Deployments: []string{},
			}
		}
		for _, deployer := range o.Deployer.EnabledDeployers {
			deploymentName := deployer + "-" + o.Config.DeployerManagement.Agent.Name + "-" + deployer + "-deployer"
			o.Config.LsDeployments.AdditionalDeployments.Deployments = append(o.Config.LsDeployments.AdditionalDeployments.Deployments, deploymentName)
		}
	}

	if err := healthcheck.AddControllersToManager(ctx, ctrlLogger, hostMgr, o.Config.LsDeployments); err != nil {
		return fmt.Errorf("unable to register health check controller: %w", err)
	}

	if err := deployitemctrl.AddControllerToManager(ctrlLogger,
		lsMgr,
		o.Config.Controllers.DeployItems,
		o.Config.DeployItemTimeouts.Pickup,
		o.Config.DeployItemTimeouts.ProgressingDefault); err != nil {
		return fmt.Errorf("unable to setup deployitem controller: %w", err)
	}

	if err := targetsync.AddControllerToManagerForTargetSyncs(ctrlLogger, lsMgr); err != nil {
		return fmt.Errorf("unable to register target sync controller: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		lockCleaner := lock.NewLockCleaner(lsMgr.GetClient(), hostMgr.GetClient())
		lockCleaner.StartPeriodicalSyncObjectCleanup(ctx, ctrlLogger)
		return nil
	})

	eg.Go(func() error {
		monitor := monitoring.NewMonitor(lsutils.GetCurrentPodNamespace(), hostMgr.GetClient())
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

// DeployInternalDeployers automatically deploys configured deployers using the new Deployer registrations.
func (o *Options) DeployInternalDeployers(ctx context.Context, mgr manager.Manager) error {
	directClient, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct client: %q", err)
	}
	ctx = logging.NewContext(ctx, logging.Wrap(ctrl.Log.WithName("deployerManagement")))
	return o.Deployer.DeployInternalDeployers(ctx, directClient, o.Config)
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
