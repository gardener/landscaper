package cd_facade

import (
	"io"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Cache interface {
	io.Closer
	Get(desc ocispecv1.Descriptor) (io.ReadCloser, error)
	Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error
}

func NewCacheBasic(log logr.Logger) (Cache, error) {
	return cache.NewCache(log)
}

func NewCache(log logr.Logger, cfg *config.OCICacheConfiguration, uid string) (Cache, error) {
	options := toOCICacheOptions(cfg, uid)
	return cache.NewCache(log, options...)
}

// ToOCICacheOptions converts a landscaper cache configuration to the cache internal config
func toOCICacheOptions(cfg *config.OCICacheConfiguration, uid string) []cache.Option {
	cacheOpts := make([]cache.Option, 0)
	if cfg != nil {
		if len(cfg.Path) != 0 {
			cacheOpts = append(cacheOpts, cache.WithBasePath(cfg.Path))
		}
		cacheOpts = append(cacheOpts, cache.WithInMemoryOverlay(cfg.UseInMemoryOverlay))
	}
	cacheOpts = append(cacheOpts, cache.WithUID(uid))
	return cacheOpts
}
