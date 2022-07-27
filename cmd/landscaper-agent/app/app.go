// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/gardener/landscaper/pkg/agent"

	"github.com/gardener/landscaper/pkg/version"
)

func NewLandscaperAgentCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "landscaper-agent",
		Short: "Landscaper agent manages a environment with deployers",

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
	o.log.Info(fmt.Sprintf("Start Landscaper Agent with version %q", version.Get().String()))

	if err := agent.AddToManager(ctx, o.log.Logr(), o.LsMgr, o.HostMgr, o.config); err != nil {
		return fmt.Errorf("unable to setup default agent: %w", err)
	}

	o.log.Info("Starting the controllers")
	eg, ctx := errgroup.WithContext(ctx)

	if o.LsMgr != o.HostMgr {
		eg.Go(func() error {
			if err := o.HostMgr.Start(ctx); err != nil {
				return fmt.Errorf("error while running host manager: %w", err)
			}
			return nil
		})
		o.log.Info("Waiting for host cluster cache to sync")
		if !o.HostMgr.GetCache().WaitForCacheSync(ctx) {
			return errors.New("unable to sync host cluster cache")
		}
		o.log.Info("Cache of host cluster successfully synced")
	}
	eg.Go(func() error {
		if err := o.LsMgr.Start(ctx); err != nil {
			return fmt.Errorf("error while running landscaper manager: %w", err)
		}
		return nil
	})
	return eg.Wait()
}
