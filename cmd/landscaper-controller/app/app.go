// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	coctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/componentoverwrites"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	install "github.com/gardener/landscaper/apis/core/install"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	deployitemctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	executionactrl "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	installationsctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/version"

	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
)

func NewLandscaperControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "landscaper-controller",
		Short: "Landscaper controller manages the orchestration of components",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(ctx); err != nil {
				options.log.Error(err, "unable to run landscaper controller")
				os.Exit(1)
			}
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.log.Info(fmt.Sprintf("Start Landscaper Controller with version %q", version.Get().String()))

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
	}

	if o.config.Metrics != nil {
		opts.MetricsBindAddress = fmt.Sprintf(":%d", o.config.Metrics.Port)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	componentcliMetrics.RegisterCacheMetrics(controllerruntimeMetrics.Registry)

	crdmgr, err := crdmanager.NewCrdManager(ctrl.Log.WithName("setup").WithName("CRDManager"), mgr, o.config)
	if err != nil {
		return fmt.Errorf("unable to setup CRD manager: %w", err)
	}

	if err := crdmgr.EnsureCRDs(); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	install.Install(mgr.GetScheme())

	componentOverwriteMgr := componentoverwrites.New()
	if err := coctrl.AddControllerToManager(mgr, componentOverwriteMgr); err != nil {
		return fmt.Errorf("unable to setup commponent overwrites controller: %w", err)
	}

	if err := installationsctrl.AddControllerToManager(mgr, componentOverwriteMgr, o.config); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactrl.AddControllerToManager(mgr); err != nil {
		return fmt.Errorf("unable to setup execution controller: %w", err)
	}

	if err := deployitemctrl.AddControllerToManager(mgr, o.config.DeployItemTimeouts.Pickup, o.config.DeployItemTimeouts.Abort, o.config.DeployItemTimeouts.ProgressingDefault); err != nil {
		return fmt.Errorf("unable to setup deployitem controller: %w", err)
	}

	for _, deployerName := range o.deployer.EnabledDeployers {
		o.log.Info("Enable Deployer", "name", deployerName)
		if deployerName == "container" {
			config := &containerv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			config.OCI = o.config.Registry.OCI
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			containerctlr.DefaultConfiguration(config)
			if err := containerctlr.AddControllerToManager(mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add container deployer: %w", err)
			}
		} else if deployerName == "helm" {
			config := &helmv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			config.OCI = o.config.Registry.OCI
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			if err := helmctlr.AddControllersToManager(mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "manifest" {
			config := &manifestv1alpha2.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			if err := manifestctlr.AddControllerToManager(mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "mock" {
			config := &mockv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			if err := mockctlr.AddControllerToManager(mgr, config); err != nil {
				return fmt.Errorf("unable to add mock deployer: %w", err)
			}
		} else {
			return fmt.Errorf("unknown deployer %s", deployerName)
		}
	}

	o.log.Info("starting the controllers")
	if err := mgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}
