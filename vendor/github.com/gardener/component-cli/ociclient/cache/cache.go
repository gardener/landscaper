// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient/metrics"
)

type layeredCache struct {
	log logr.Logger
	mux sync.RWMutex

	baseFs    *FileSystem
	overlayFs *FileSystem
}

// NewCache creates a new cache with the given options.
// It uses by default a tmp fs
func NewCache(log logr.Logger, options ...Option) (*layeredCache, error) {
	opts := &Options{}
	opts = opts.ApplyOptions(options)
	opts.ApplyDefaults()

	if err := initBasePath(opts); err != nil {
		return nil, err
	}

	base, err := projectionfs.New(osfs.New(), opts.BasePath)
	if err != nil {
		return nil, err
	}
	baseCFs, err := NewCacheFilesystem(log.WithName("baseCacheFS"), base, opts.BaseGCConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create base layer: %w", err)
	}
	var overlayCFs *FileSystem
	if opts.InMemoryOverlay {
		overlayCFs, err = NewCacheFilesystem(log.WithName("inMemoryCacheFS"), memoryfs.New(), opts.InMemoryGCConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to create base layer: %w", err)
		}
	}

	//initialize metrics
	baseCFs.WithMetrics(
		metrics.CachedItems.WithLabelValues(opts.UID),
		metrics.CacheDiskUsage.WithLabelValues(opts.UID),
		metrics.CacheHitsDisk.WithLabelValues(opts.UID))
	if opts.InMemoryOverlay {
		overlayCFs.WithMetrics(nil,
			metrics.CacheMemoryUsage.WithLabelValues(opts.UID),
			metrics.CacheHitsMemory.WithLabelValues(opts.UID))
	}

	return &layeredCache{
		log:       log,
		mux:       sync.RWMutex{},
		baseFs:    baseCFs,
		overlayFs: overlayCFs,
	}, nil
}

func initBasePath(opts *Options) error {
	if len(opts.BasePath) == 0 {
		path, err := ioutil.TempDir(os.TempDir(), "ocicache")
		if err != nil {
			return err
		}
		opts.BasePath = path
	}
	info, err := os.Stat(opts.BasePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(opts.BasePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if !info.IsDir() {
		return errors.New("path has to be a directory")
	}
	return nil
}

// Close implements the io.Closer interface that cleanups all resource used by the cache.
func (lc *layeredCache) Close() error {
	if err := lc.baseFs.Close(); err != nil {
		return err
	}
	if lc.overlayFs != nil {
		if err := lc.overlayFs.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (lc *layeredCache) Get(desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	_, file, err := lc.get(Path(desc), desc)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (lc *layeredCache) Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error {
	path := Path(desc)
	lc.mux.Lock()
	defer lc.mux.Unlock()
	defer reader.Close()

	file, err := lc.baseFs.Create(path, desc.Size)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (lc *layeredCache) Info() (Info, error) {
	return Info{
		Size:        lc.baseFs.Size,
		CurrentSize: lc.baseFs.CurrentSize(),
		ItemsCount:  int64(lc.baseFs.index.Len()),
	}, nil
}

func (lc *layeredCache) Prune() error {
	return lc.baseFs.DeleteAll()
}

func (lc *layeredCache) get(dgst string, desc ocispecv1.Descriptor) (os.FileInfo, vfs.File, error) {
	lc.mux.RLock()
	defer lc.mux.RUnlock()

	// first search in the overlayFs layer
	if info, file, err := lc.getFromOverlay(dgst, desc); err == nil {
		return info, file, nil
	}

	info, err := lc.baseFs.Stat(dgst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	verified, err := verifyBlob(lc.baseFs.FileSystem, info, dgst, desc)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to verify blob: %w", err)
	}
	if !verified {
		// remove invalid blob from cache
		if err := lc.baseFs.Remove(dgst); err != nil {
			lc.log.V(7).Info("unable to remove invalid blob", "digest", dgst, "err", err.Error())
		}
		return info, nil, ErrNotFound
	}
	file, err := lc.baseFs.OpenFile(dgst, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	// copy file to in memory cache
	if lc.overlayFs != nil {
		overlayFile, err := lc.overlayFs.Create(dgst, info.Size())
		if err != nil {
			// do not return an error here as we are only unable to write to better cache
			lc.log.V(5).Info(err.Error())
			return info, file, nil
		}
		defer overlayFile.Close()
		if _, err := io.Copy(overlayFile, file); err != nil {
			// do not return an error here as we are only unable to write to better cache
			lc.log.V(5).Info(err.Error())

			// The file handle is at the end as the data was copied by io.Copy.
			// Therefore the file handle is reset so that the caller can also read the data.
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				return nil, nil, fmt.Errorf("unable to reset the file handle: %w", err)
			}
			return info, file, nil
		}

		// The file handle is at the end as the data was copied by io.Copy.
		// Therefore the file handle is reset so that the caller can also read the data.
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, nil, fmt.Errorf("unable to reset the file handle: %w", err)
		}
	}
	return info, file, nil
}

func (lc *layeredCache) getFromOverlay(dgst string, desc ocispecv1.Descriptor) (os.FileInfo, vfs.File, error) {
	if lc.overlayFs == nil {
		return nil, nil, ErrNotFound
	}
	info, err := lc.overlayFs.Stat(dgst)
	if err != nil {
		lc.log.V(7).Info("not found in overlay cache", "digest", dgst, "err", err.Error())
		return nil, nil, ErrNotFound
	}

	verified, err := verifyBlob(lc.overlayFs.FileSystem, info, dgst, desc)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to verify blob: %w", err)
	}
	if !verified {
		// remove invalid blob from cache
		if err := lc.overlayFs.Remove(dgst); err != nil {
			lc.log.V(7).Info("unable to remove invalid blob", "digest", dgst, "err", err.Error())
		}
		return info, nil, ErrNotFound
	}
	file, err := lc.overlayFs.OpenFile(dgst, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}
	return info, file, err
}

// verifyBlob validates the digest of a blob
func verifyBlob(fs vfs.FileSystem, info os.FileInfo, dgst string, desc ocispecv1.Descriptor) (bool, error) {
	if info.Size() != desc.Size {
		// do a simple check by checking the blob size
		return false, nil
	}

	file, err := fs.OpenFile(dgst, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer file.Close()

	verifier := desc.Digest.Verifier()
	if _, err := io.Copy(verifier, file); err != nil {
		return false, err
	}
	return verifier.Verified(), nil
}

func Path(desc ocispecv1.Descriptor) string {
	return desc.Digest.Encoded()
}
