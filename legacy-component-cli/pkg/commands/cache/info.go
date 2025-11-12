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
	"sigs.k8s.io/yaml"

	cache2 "github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

type InfoOptions struct{}

func NewInfoCommand(ctx context.Context) *cobra.Command {
	opts := &InfoOptions{}
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Shows info about the currently used cache",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}
	return cmd
}

func (o *InfoOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
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

	type extendedCacheInfo struct {
		Size        string `json:"Size,omitempty"`
		CurrentSize string `json:"CurrentSize"`
		ItemsCount  int64  `json:"Items"`
		Usage       string `json:"Usage,omitempty"`
	}
	eInfo := extendedCacheInfo{
		CurrentSize: utils.BytesString(uint64(info.CurrentSize), 2),
		ItemsCount:  info.ItemsCount,
	}
	if info.Size != 0 {
		eInfo.Size = utils.BytesString(uint64(info.Size), 2)
	}
	if info.Size != 0 && info.CurrentSize != 0 {
		usage := float64(info.Size) / float64(info.CurrentSize) * 100
		eInfo.Usage = fmt.Sprintf("%f%%", usage)
	}

	infoBytes, err := yaml.Marshal(eInfo)
	if err != nil {
		return fmt.Errorf("unable to marshal cache info: %w", err)
	}
	fmt.Printf("Cache Info from %s\n\n", cacheDir)
	fmt.Println(string(infoBytes))
	return nil
}
