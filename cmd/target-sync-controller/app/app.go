// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	lsutils "github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/landscaper/controllers/targetsync"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsinstall "github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/pkg/version"
)

// NewTargetSyncControllerCommand creates a new command for the landscaper service controller
func NewTargetSyncControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "target-sync-controller",
		Short: "Target sync controller syncronises secrets into custom namespaces and creates corresponding targets",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(ctx); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(ctx); err != nil {
				options.Log.Error(err, "unable to run target sync controller")
			}
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.Log.Info(fmt.Sprintf("Start Target Sync Controller with version %q", version.Get().String()))

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
		NewClient:          lsutils.NewUncachedClient,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	lsinstall.Install(mgr.GetScheme())

	ctrlLogger := o.Log.WithName("controllers")
	if err := targetsync.AddControllerToManagerForTargetSyncs(ctrlLogger, mgr); err != nil {
		return fmt.Errorf("unable to setup landscaper deployments controller: %w", err)
	}

	o.Log.Info("starting the controllers")
	if err := mgr.Start(ctx); err != nil {
		o.Log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}
