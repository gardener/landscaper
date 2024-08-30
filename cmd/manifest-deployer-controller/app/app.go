// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/spf13/cobra"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

func NewManifestDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:          "manifest-deployer",
		Short:        fmt.Sprintf("Manifest Deployer is a controller that applies kubernetes manifests based on DeployItems of type %s", manifestctlr.Type),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Complete(); err != nil {
				return err
			}
			return options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.DeployerOptions.Log.Info("Starting Manifest Deployer", lc.KeyVersion, version.Get().GitVersion)
	if err := manifestctlr.AddDeployerToManager(
		o.DeployerOptions.LsUncachedClient, o.DeployerOptions.LsCachedClient, o.DeployerOptions.HostUncachedClient, o.DeployerOptions.HostCachedClient,
		o.DeployerOptions.FinishedObjectCache,
		o.DeployerOptions.Log, o.DeployerOptions.LsMgr,
		o.DeployerOptions.HostMgr, o.Config, "manifest"); err != nil {
		return fmt.Errorf("unable to setup manifest controller")
	}

	if os.Getenv("ENABLE_PROFILER") == "true" {
		go func() {
			o.DeployerOptions.Log.Info("Starting profiler for manifest deployer")
			err := http.ListenAndServe("localhost:8081", nil)
			o.DeployerOptions.Log.Error(err, "manifest deployer profiler stopped")
		}()

		go utils.LogMemStatsPeriodically(logging.NewContext(ctx, o.DeployerOptions.Log), 60*time.Second,
			o.DeployerOptions.HostUncachedClient, "manifest-deployer")
	}

	o.DeployerOptions.Log.Info("Starting manifest deployer manager")
	return o.DeployerOptions.StartManagers(ctx)
}
