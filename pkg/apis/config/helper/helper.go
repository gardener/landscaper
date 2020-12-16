// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"

	"github.com/gardener/landscaper/pkg/apis/config"
)

// WithConfiguration applies external oci configuration as internal options.
type WithConfigurationStruct config.OCIConfiguration

func (c *WithConfigurationStruct) ApplyOption(options *ociclient.Options) {
	if c == nil {
		return
	}
	if len(c.ConfigFiles) != 0 {
		options.Paths = c.ConfigFiles
	}
	if c.Cache != nil {
		options.CacheConfig = &cache.Options{
			InMemoryOverlay: c.Cache.UseInMemoryOverlay,
			BasePath:        c.Cache.Path,
		}
	}
	options.AllowPlainHttp = c.AllowPlainHttp
}

// WithConfiguration applies external oci configuration as internal options.
func WithConfiguration(cfg *config.OCIConfiguration) *WithConfigurationStruct {
	if cfg == nil {
		return nil
	}
	wc := WithConfigurationStruct(*cfg)
	return &wc
}

// ToOCICacheOptions converts a landscaper cache configuration to the cache internal config
func ToOCICacheOptions(cfg *config.OCICacheConfiguration) []cache.Option {
	cacheOpts := make([]cache.Option, 0)
	if cfg != nil {
		if len(cfg.Path) != 0 {
			cacheOpts = append(cacheOpts, cache.WithBasePath(cfg.Path))
		}
		cacheOpts = append(cacheOpts, cache.WithInMemoryOverlay(cfg.UseInMemoryOverlay))
	}
	return cacheOpts
}
