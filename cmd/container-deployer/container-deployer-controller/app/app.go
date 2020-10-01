// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/apis/core/install"
	containeractuator "github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/version"
)

func NewContainerDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:     "container-deployer",
		Short:   "Container Deployer is a controller that executes containers based on DeployItems of type Container",
		Version: version.Get().GitVersion,
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

	if err := containeractuator.AddActuatorToManager(hostClusterMgr, landscaperClusterMgr, o.config); err != nil {
		o.log.Error(err, "unable to setup controller")
		os.Exit(1)
	}

	o.log.Info(fmt.Sprintf("starting the controllers with version %s", version.Get().GitVersion))

	if landscaperClusterMgr != hostClusterMgr {
		go func() {
			if err := hostClusterMgr.Start(ctx.Done()); err != nil {
				o.log.Error(err, "error while running manager")
				os.Exit(1)
			}
		}()
		o.log.Info("Waiting for host cluster cache to sync")
		if !hostClusterMgr.GetCache().WaitForCacheSync(ctx.Done()) {
			o.log.Info("Unable to sync host cluster cache")
			os.Exit(1)
		}

		o.log.Info("Cache of host cluster successfully synced")
	}
	if err := landscaperClusterMgr.Start(ctx.Done()); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}

}
