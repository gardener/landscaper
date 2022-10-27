// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"k8s.io/client-go/tools/clientcmd"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	lsutils "github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/landscaper/controllers/targetsync"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsconfig "github.com/gardener/landscaper/apis/config"
	lsinstall "github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/pkg/version"

	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
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
	setupLogger := o.Log.WithName("setup for target sync controller")
	setupLogger.Info("Starting TargetSync Controller", lc.KeyVersion, version.Get().String())

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
		NewClient:          lsutils.NewUncachedClient,
	}

	data, err := os.ReadFile(o.landscaperKubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to read landscaper kubeconfig for target sync controller from %s: %w", o.landscaperKubeconfigPath, err)
	}

	client, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return fmt.Errorf("unable to build landscaper cluster client for target sync controller from %s: %w", o.landscaperKubeconfigPath, err)
	}

	lsConfig, err := client.ClientConfig()
	if err != nil {
		return fmt.Errorf("unable to build landscaper cluster rest client for target sync controller from %s: %w", o.landscaperKubeconfigPath, err)
	}

	lsMgr, err := ctrl.NewManager(lsConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager for target sync controller: %w", err)
	}

	if o.installCrd {
		if err := o.ensureCRDs(ctx, lsMgr); err != nil {
			return err
		}
	}

	lsinstall.Install(lsMgr.GetScheme())

	ctrlLogger := o.Log.WithName("controllers")
	if err := targetsync.AddControllerToManagerForTargetSyncs(ctrlLogger, lsMgr); err != nil {
		return fmt.Errorf("unable to setup landscaper deployments controller for target sync controller: %w", err)
	}

	o.Log.Info("starting the controller for target sync")
	if err := lsMgr.Start(ctx); err != nil {
		o.Log.Error(err, "error while running manager for target sync controller")
		os.Exit(1)
	}
	return nil
}

func (o *options) ensureCRDs(ctx context.Context, mgr manager.Manager) error {
	ctx = logging.NewContext(ctx, logging.Wrap(ctrl.Log.WithName("crdManager")))
	crdConfig := lsconfig.CrdManagementConfiguration{
		DeployCustomResourceDefinitions: pointer.Bool(true),
		ForceUpdate:                     pointer.Bool(true),
	}

	lsConfig := lsconfig.LandscaperConfiguration{
		CrdManagement: crdConfig,
	}
	crdmgr, err := crdmanager.NewCrdManager(mgr, &lsConfig)

	if err != nil {
		return fmt.Errorf("unable to setup CRD manager: %w", err)
	}

	if err := crdmgr.EnsureCRDs(ctx); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	return nil
}
