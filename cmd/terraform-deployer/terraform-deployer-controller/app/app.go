// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/landscaper/apis/core/install"
	terraformctlr "github.com/gardener/landscaper/pkg/deployer/terraform"
)

func NewTerraformDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "terraform-deployer",
		Short: "Terraform Deployer is a controller that deploys terraform configuration based on DeployItems of type Terraform",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) {
	opts := manager.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0", // disable the metrics serving by default
	}

	hostClusterMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		o.log.Error(err, "unable to setup manager for host cluster")
		os.Exit(1)
	}

	landscaperClusterMgr := hostClusterMgr
	if o.landscaperClusterRestConfig != nil {
		landscaperClusterMgr, err = ctrl.NewManager(o.landscaperClusterRestConfig, opts)
		if err != nil {
			o.log.Error(err, "unable to setup manager for landscaper cluster")
			os.Exit(1)
		}
	}
	install.Install(landscaperClusterMgr.GetScheme())

	if err := terraformctlr.AddControllerToManager(hostClusterMgr, landscaperClusterMgr, o.config); err != nil {
		o.log.Error(err, "unable to setup controller")
		os.Exit(1)
	}

	if landscaperClusterMgr != hostClusterMgr {
		go func() {
			if err := hostClusterMgr.Start(ctx); err != nil {
				o.log.Error(err, "error while running manager")
				os.Exit(1)
			}
		}()
		o.log.Info("Waiting for host cluster cache to sync")
		if !hostClusterMgr.GetCache().WaitForCacheSync(ctx) {
			o.log.Info("Unable to sync host cluster cache")
			os.Exit(1)
		}

		o.log.Info("Cache of host cluster successfully synced")
	}
	if err := landscaperClusterMgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
}
