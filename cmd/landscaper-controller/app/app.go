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
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/landscaper/blueprints"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	coctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/componentoverwrites"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	"github.com/gardener/landscaper/pkg/agent"

	deployers "github.com/gardener/landscaper/pkg/deployermanagement/controller"

	install "github.com/gardener/landscaper/apis/core/install"
	deployitemctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	executionactrl "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	installationsctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/version"

	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
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
	o.Log.Info(fmt.Sprintf("Start Landscaper Controller with version %q", version.Get().String()))

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
	}

	if o.Config.Metrics != nil {
		opts.MetricsBindAddress = fmt.Sprintf(":%d", o.Config.Metrics.Port)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	componentcliMetrics.RegisterCacheMetrics(controllerruntimeMetrics.Registry)

	store, err := blueprints.NewStore(o.Log, osfs.New(), o.Config.BlueprintStore)
	if err != nil {
		return fmt.Errorf("unable to setup blueprint store: %w", err)
	}
	blueprints.SetStore(store)

	crdmgr, err := crdmanager.NewCrdManager(ctrl.Log.WithName("setup").WithName("CRDManager"), mgr, o.Config)
	if err != nil {
		return fmt.Errorf("unable to setup CRD manager: %w", err)
	}

	if err := crdmgr.EnsureCRDs(ctx); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	install.Install(mgr.GetScheme())

	ctrlLogger := o.Log.WithName("controllers")
	componentOverwriteMgr := componentoverwrites.New()
	if err := coctrl.AddControllerToManager(ctrlLogger, mgr, componentOverwriteMgr); err != nil {
		return fmt.Errorf("unable to setup commponent overwrites controller: %w", err)
	}

	if err := installationsctrl.AddControllerToManager(ctrlLogger, mgr, componentOverwriteMgr, o.Config); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactrl.AddControllerToManager(ctrlLogger, mgr); err != nil {
		return fmt.Errorf("unable to setup execution controller: %w", err)
	}

	if err := deployitemctrl.AddControllerToManager(ctrlLogger, mgr, o.Config.DeployItemTimeouts.Pickup, o.Config.DeployItemTimeouts.Abort, o.Config.DeployItemTimeouts.ProgressingDefault); err != nil {
		return fmt.Errorf("unable to setup deployitem controller: %w", err)
	}

	if !o.Config.DeployerManagement.Disable {
		if err := deployers.AddControllersToManager(ctrlLogger, mgr, o.Config); err != nil {
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
						{
							Key:      lsv1alpha1.NotUseDefaultDeployerAnnotation,
							Operator: selection.Exists,
						},
					},
				},
			)
			if err := agent.AddToManager(ctx, o.Log, mgr, mgr, agentConfig); err != nil {
				return fmt.Errorf("unable to setup default agent: %w", err)
			}
		}
		if err := o.DeployInternalDeployers(ctx, mgr); err != nil {
			return err
		}
	}

	o.Log.Info("starting the controllers")
	if err := mgr.Start(ctx); err != nil {
		o.Log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}

// DeployInternalDeployers automatically deploys configured deployers using the new Deployer registrations.
func (o *Options) DeployInternalDeployers(ctx context.Context, mgr manager.Manager) error {
	directClient, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct client: %q", err)
	}
	return o.Deployer.DeployInternalDeployers(ctx, o.Log, directClient, o.Config)
}
