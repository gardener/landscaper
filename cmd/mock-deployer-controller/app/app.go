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

	"github.com/gardener/landscaper/pkg/apis/core/install"
	mockactuator "github.com/gardener/landscaper/pkg/deployer/mock"
)

func NewMockDeployerControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "mock-deployer",
		Short: "Mock Deployer is a controller that mocks the behavior of deploy items",

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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		o.log.Error(err, "unable to setup manager")
		os.Exit(1)
	}

	install.Install(mgr.GetScheme())

	if err := mockactuator.AddActuatorToManager(mgr); err != nil {
		o.log.Error(err, "unable to setup controller")
		os.Exit(1)
	}

	o.log.Info("starting the controller")
	if err := mgr.Start(ctx.Done()); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
}
