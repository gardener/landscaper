// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cachecmd

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"

	cache2 "github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

// PruneOptions describes the options for pruning the cache
type PruneOptions struct{}

// NewPruneCommand creates a new prune cache command
func NewPruneCommand(ctx context.Context) *cobra.Command {
	opts := &PruneOptions{}
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prunes all currently cached files",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}
	return cmd
}

func (o *PruneOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	cacheDir, err := utils.CacheDir()
	if err != nil {
		return fmt.Errorf("unable to get oci cache directory: %w", err)
	}

	cache, err := cache2.NewCache(log, cache2.WithBasePath(cacheDir))
	if err != nil {
		return err
	}
	info, err := cache.Info()
	if err != nil {
		return err
	}
	if err := cache.Prune(); err != nil {
		return err
	}

	fmt.Printf("Successfully pruned %d items from the cache %s\n", info.ItemsCount, cacheDir)
	return nil
}
