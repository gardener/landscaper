// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsconfig "github.com/gardener/landscaper/apis/config"
	lsinstall "github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/targetsync"
	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
	lsutils "github.com/gardener/landscaper/pkg/utils"
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
	o.Log.Info("Starting TargetSync Controller", lc.KeyVersion, version.Get().GitVersion)

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
		Cache:              cache.Options{SyncPeriod: ptr.To[time.Duration](time.Hour * 24 * 1000)},
	}

	data, err := os.ReadFile(o.landscaperKubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to read landscaper kubeconfig for target sync controller from %s: %w", o.landscaperKubeconfigPath, err)
	}

	lsRestConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return fmt.Errorf("unable to build landscaper cluster rest client for target sync controller from %s: %w", o.landscaperKubeconfigPath, err)
	}
	lsRestConfig = lsutils.RestConfigWithModifiedClientRequestRestrictions(o.Log, lsRestConfig, 10, 8)

	lsMgr, err := ctrl.NewManager(lsRestConfig, opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager for target sync controller: %w", err)
	}

	if o.installCrd {
		if err := o.ensureCRDs(ctx, lsMgr); err != nil {
			return err
		}
	}

	lsinstall.Install(lsMgr.GetScheme())

	lsUncachedClient, err := lsutils.NewUncachedClientFromManager(lsMgr)
	if err != nil {
		return fmt.Errorf("unable to build new uncached ls client: %w", err)
	}
	lsCachedClient := lsMgr.GetClient()

	if err := targetsync.AddControllerToManagerForTargetSyncs(lsUncachedClient, lsCachedClient, o.Log, lsMgr); err != nil {
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
		DeployCustomResourceDefinitions: ptr.To[bool](true),
		ForceUpdate:                     ptr.To[bool](true),
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
