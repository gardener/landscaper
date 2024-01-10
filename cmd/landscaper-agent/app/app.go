// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/agent"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/version"
)

func NewLandscaperAgentCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "landscaper-agent",
		Short: "Landscaper agent manages a environment with deployers",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				panic(err)
			}
			if err := options.run(ctx); err != nil {
				panic(fmt.Errorf("unable to run landscaper controller: %w", err))
			}
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.log.Info("Starting Landscaper Agent", lc.KeyVersion, version.Get().String())

	lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient, err := lsutils.ClientsFromManagers(o.LsMgr, o.HostMgr)
	if err != nil {
		return err
	}

	if err := agent.AddToManager(ctx, lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient,
		o.log, o.LsMgr, o.HostMgr, o.config, "agent-helm"); err != nil {
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
