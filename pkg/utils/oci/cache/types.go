// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"errors"
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/apis/config"
)

var (
	// ErrNotFound is a error that indicates that the file is not cached
	ErrNotFound = errors.New("not cached")
)

// Cache is the interface for a oci cache
type Cache interface {
	Get(desc ocispecv1.Descriptor) (io.ReadCloser, error)
	Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error
}

// InjectCache is a interface to inject a cache.
type InjectCache interface {
	InjectCache(c Cache) error
}

// InjectCacheInto injects a cache if the given object implements the InjectCache interface.
func InjectCacheInto(obj interface{}, cache Cache) error {
	if cache == nil {
		return nil
	}
	if injector, ok := obj.(InjectCache); ok {
		return injector.InjectCache(cache)
	}
	return nil
}

// Options contains all oci cache options to configure the oci cache.
type Options struct {
	// InMemoryOverlay specifies if a overlayFs InMemory cache should be used
	InMemoryOverlay bool

	// OverlaySizeMB is the size of the overlay cache in MB
	OverlaySizeMB int64

	// BasePath specifies the Base path for the os filepath.
	// Will be defaulted to a temp filepath if not specified
	BasePath string

	// BaseSizeGB is the max size of the base cache in GB
	BaseSizeGB int64
}

// Option is the interface to specify different cache options
type Option interface {
	ApplyOption(options *Options)
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		opt.ApplyOption(o)
	}
	return o
}

// WithConfiguration applies configuration based on a external configuration.
func WithConfiguration(cfg *config.OCICacheConfiguration) *WithConfigurationStruct {
	if cfg == nil {
		return nil
	}
	wc := WithConfigurationStruct(*cfg)
	return &wc
}

// WithConfiguration applies configuration based on a external configuration.
type WithConfigurationStruct config.OCICacheConfiguration

func (w *WithConfigurationStruct) ApplyOption(options *Options) {
	if w == nil {
		return
	}
	if w.UseInMemoryOverlay {
		options.InMemoryOverlay = w.UseInMemoryOverlay
	}
	if len(w.Path) != 0 {
		options.BasePath = w.Path
	}
}

// WithInMemoryOverlay is the options to specify the usage of a in memory overlayFs
type WithInMemoryOverlay bool

func (w WithInMemoryOverlay) ApplyOption(options *Options) {
	options.InMemoryOverlay = bool(w)
}

// WithBasePath is the options to specify a base path
type WithBasePath string

func (p WithBasePath) ApplyOption(options *Options) {
	options.BasePath = string(p)
}

// WithOverlaySize sets the max size of the overly file system in MB
type WithOverlaySize int64

func (p WithOverlaySize) ApplyOption(options *Options) {
	options.OverlaySizeMB = int64(p)
}

// WithBaseSize sets the max size of the base file system in GB
type WithBaseSize int64

func (p WithBaseSize) ApplyOption(options *Options) {
	options.BaseSizeGB = int64(p)
}
