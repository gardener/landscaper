// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/marstr/guid"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/iotools"
	"github.com/open-component-model/ocm/pkg/refmgmt"
)

const (
	BLOB_UNKNOWN_SIZE   = blobaccess.BLOB_UNKNOWN_SIZE
	BLOB_UNKNOWN_DIGEST = blobaccess.BLOB_UNKNOWN_DIGEST
)

type StaticAllocatable struct{}

func (_ StaticAllocatable) Ref() error   { return nil }
func (_ StaticAllocatable) Unref() error { return nil }

type BlobSource interface {
	refmgmt.Allocatable
	GetBlobData(digest digest.Digest) (int64, blobaccess.DataAccess, error)
}

type BlobSink interface {
	refmgmt.Allocatable
	AddBlob(blob blobaccess.BlobAccess) (int64, digest.Digest, error)
}

type RootedCache interface {
	Root() (string, vfs.FileSystem)
}

type CleanupCache interface {
	// Cleanup can be implemented to offer a cache reorg.
	// It returns the number and size of
	//	- handled entries (cnt, size)
	//	- not handled entries (ncnt, nsize)
	//	- failing entries (fcnt, fsize)
	Cleanup(p common.Printer, before *time.Time, dryrun bool) (cnt int, ncnt int, fcnt int, size int64, nsize int64, fsize int64, err error)
}

type BlobCache interface {
	BlobSource
	BlobSink
	AddData(data blobaccess.DataAccess) (int64, digest.Digest, error)
}

type blobCache struct {
	refmgmt.Allocatable
	lock  sync.RWMutex
	cache vfs.FileSystem
}

var (
	_ sync.Locker = (*blobCache)(nil)
	_ RootedCache = (*blobCache)(nil)
)

// ACCESS_SUFFIX is the suffix of an additional blob related
// file used to track the last access time by its modification time,
// because Go does not support a platform independent way to access the
// last access time attribute of a filesystem.
const ACCESS_SUFFIX = ".acc"

func NewDefaultBlobCache(fss ...vfs.FileSystem) (BlobCache, error) {
	var err error
	fs := DefaultedFileSystem(nil, fss...)
	if fs == nil {
		fs, err = osfs.NewTempFileSystem()
		if err != nil {
			return nil, err
		}
	}
	c := &blobCache{
		cache: fs,
	}
	c.Allocatable = refmgmt.NewAllocatable(c.cleanup)
	return c, nil
}

func NewStaticBlobCache(path string, fss ...vfs.FileSystem) (BlobCache, error) {
	fs := FileSystem(fss...)
	err := fs.MkdirAll(path, 0o700)
	if err != nil {
		return nil, err
	}
	fs, err = projectionfs.New(fs, path)
	if err != nil {
		return nil, err
	}
	return NewDefaultBlobCache(fs)
}

func (c *blobCache) Root() (string, vfs.FileSystem) {
	return vfs.PathSeparatorString, c.cache
}

func (c *blobCache) Lock() {
	c.lock.Lock()
}

func (c *blobCache) Unlock() {
	c.lock.Unlock()
}

func (c *blobCache) Cleanup(p common.Printer, before *time.Time, dryrun bool) (cnt int, ncnt int, fcnt int, size int64, nsize int64, fsize int64, err error) {
	c.Lock()
	defer c.Unlock()

	if p == nil {
		p = common.NewPrinter(nil)
	}
	path, fs := c.Root()

	entries, err := vfs.ReadDir(fs, path)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ACCESS_SUFFIX) {
			continue
		}
		base := vfs.Join(fs, path, e.Name())
		if before != nil && !before.IsZero() {
			fi, err := fs.Stat(base + ACCESS_SUFFIX)
			if err != nil {
				if !vfs.IsErrNotExist(err) {
					if p != nil {
						p.Printf("cannot stat %q: %s", e.Name(), err)
					}
					fcnt++
					fsize += e.Size()
					continue
				}
			} else {
				if fi.ModTime().After(*before) {
					ncnt++
					nsize += e.Size()
					continue
				}
			}
		}
		if !dryrun {
			err := fs.RemoveAll(base)
			if err != nil {
				if p != nil {
					p.Printf("cannot delete %q: %s", e.Name(), err)
				}
				fcnt++
				fsize += e.Size()
				continue
			}
			fs.RemoveAll(base + ACCESS_SUFFIX)
		}
		cnt++
		size += e.Size()
	}
	return cnt, ncnt, fcnt, size, nsize, fsize, nil
}

func (c *blobCache) cleanup() error {
	return vfs.Cleanup(c.cache)
}

func (c *blobCache) GetBlobData(digest digest.Digest) (int64, blobaccess.DataAccess, error) {
	err := c.Ref()
	if err == nil {
		defer c.Unref()
		c.lock.RLock()
		defer c.lock.RUnlock()

		path := common.DigestToFileName(digest)
		fi, err := c.cache.Stat(path)
		if err == nil {
			vfs.WriteFile(c.cache, path+ACCESS_SUFFIX, []byte{}, 0o600)
			// now := time.Now()
			// c.cache.Chtimes(path+ACCESS_SUFFIX, now, now)
			return fi.Size(), blobaccess.DataAccessForFile(c.cache, path), nil
		}
		if os.IsNotExist(err) {
			return -1, nil, blobaccess.ErrBlobNotFound(digest)
		}
	}
	return BLOB_UNKNOWN_SIZE, nil, err
}

func (c *blobCache) AddBlob(blob blobaccess.BlobAccess) (int64, digest.Digest, error) {
	err := c.Ref()
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	defer c.Unref()

	var digester *iotools.DigestReader

	if blob.DigestKnown() {
		c.lock.RLock()
		path := common.DigestToFileName(blob.Digest())
		if ok, err := vfs.Exists(c.cache, path); ok || err != nil {
			c.lock.RUnlock()
			return blob.Size(), blob.Digest(), err
		}
		c.lock.RUnlock()
	}

	tmp := "TMP" + guid.NewGUID().String()

	br, err := blob.Reader()
	if err != nil {
		return BLOB_UNKNOWN_SIZE, "", errors.Wrapf(err, "cannot get blob content")
	}
	defer br.Close()

	reader := io.Reader(br)
	if !blob.DigestKnown() {
		digester = iotools.NewDefaultDigestReader(reader)
		reader = digester
	}

	writer, err := c.cache.Create(tmp)
	if err != nil {
		return BLOB_UNKNOWN_SIZE, "", errors.Wrapf(err, "cannot create blob file in cache")
	}
	defer writer.Close()
	size, err := io.Copy(writer, reader)
	if err != nil {
		c.cache.Remove(tmp)
		return BLOB_UNKNOWN_SIZE, "", err
	}

	var digest digest.Digest
	var ok bool
	if digester != nil {
		digest = digester.Digest()
	} else {
		digest = blob.Digest()
	}
	target := common.DigestToFileName(digest)

	c.lock.Lock()
	defer c.lock.Unlock()
	if ok, err = vfs.Exists(c.cache, target); err != nil || !ok {
		err = c.cache.Rename(tmp, target)
	}
	c.cache.Remove(tmp)
	vfs.WriteFile(c.cache, target+ACCESS_SUFFIX, []byte{}, 0o600)
	return size, digest, err
}

func (c *blobCache) AddData(data blobaccess.DataAccess) (int64, digest.Digest, error) {
	return c.AddBlob(blobaccess.ForDataAccess(BLOB_UNKNOWN_DIGEST, BLOB_UNKNOWN_SIZE, "", data))
}

////////////////////////////////////////////////////////////////////////////////

type cascadedCache struct {
	refmgmt.Allocatable
	lock   sync.RWMutex
	parent BlobSource
	source BlobSource
	sink   BlobSink
}

var _ BlobCache = (*cascadedCache)(nil)

func NewCascadedBlobCache(parent BlobCache) (BlobCache, error) {
	if parent != nil {
		err := parent.Ref()
		if err != nil {
			return nil, err
		}
	}
	c := &cascadedCache{
		parent: parent,
	}
	c.Allocatable = refmgmt.NewAllocatable(c.cleanup)
	return c, nil
}

func NewCascadedBlobCacheForSource(parent BlobSource, src BlobSource) (BlobCache, error) {
	if parent != nil {
		err := parent.Ref()
		if err != nil {
			return nil, err
		}
	}
	if src != nil {
		err := src.Ref()
		if err != nil {
			return nil, err
		}
	}
	c := &cascadedCache{
		parent: parent,
		source: src,
	}
	c.Allocatable = refmgmt.NewAllocatable(c.cleanup)
	return c, nil
}

func NewCascadedBlobCacheForCache(parent BlobSource, src BlobCache) (BlobCache, error) {
	if parent != nil {
		err := parent.Ref()
		if err != nil {
			return nil, err
		}
	}
	if src != nil {
		err := src.Ref()
		if err != nil {
			return nil, err
		}
	}
	c := &cascadedCache{
		parent: parent,
		source: src,
		sink:   src,
	}
	c.Allocatable = refmgmt.NewAllocatable(c.cleanup)
	return c, nil
}

func (c *cascadedCache) cleanup() error {
	list := errors.ErrListf("closing cascaded blob cache")
	if c.source != nil {
		list.Add(c.source.Unref())
	}
	if c.parent != nil {
		list.Add(c.parent.Unref())
	}
	return list.Result()
}

func (c *cascadedCache) GetBlobData(digest digest.Digest) (int64, blobaccess.DataAccess, error) {
	err := c.Ref()
	if err != nil {
		return BLOB_UNKNOWN_SIZE, nil, err
	}
	defer c.Unref()

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.source != nil {
		size, acc, err := c.source.GetBlobData(digest)
		if err == nil {
			return size, acc, err
		}
		if !blobaccess.IsErrBlobNotFound(err) {
			return BLOB_UNKNOWN_SIZE, nil, err
		}
	}
	if c.parent != nil {
		return c.parent.GetBlobData(digest)
	}
	return BLOB_UNKNOWN_SIZE, nil, blobaccess.ErrBlobNotFound(digest)
}

func (c *cascadedCache) AddData(data blobaccess.DataAccess) (int64, digest.Digest, error) {
	return c.AddBlob(blobaccess.ForDataAccess(BLOB_UNKNOWN_DIGEST, BLOB_UNKNOWN_SIZE, "", data))
}

func (c *cascadedCache) AddBlob(blob blobaccess.BlobAccess) (int64, digest.Digest, error) {
	err := c.Ref()
	if err == nil {
		defer c.Unref()
		c.lock.Lock()
		defer c.lock.Unlock()

		if c.source == nil {
			cache, err := NewDefaultBlobCache()
			if err != nil {
				return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
			}
			c.source = cache
			c.sink = cache
		}
		if c.sink != nil {
			return c.sink.AddBlob(blob)
		}
		if c.parent != nil {
			if sink, ok := c.parent.(BlobSink); ok {
				return sink.AddBlob(blob)
			}
		}
	}
	return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
}

////////////////////////////////////////////////////////////////////////////////

type cached struct {
	refmgmt.Allocatable
	lock   sync.RWMutex
	source BlobSource
	sink   BlobSink
	cache  BlobCache
}

var _ BlobCache = (*cached)(nil)

func (c *cached) cleanup() error {
	list := errors.ErrListf("closing cached blob store")
	if c.sink != nil {
		list.Add(c.sink.Unref())
	}
	if c.source != nil {
		list.Add(c.source.Unref())
	}
	c.cache.Unref()
	return list.Result()
}

func (a *cached) GetBlobData(digest digest.Digest) (int64, blobaccess.DataAccess, error) {
	err := a.Ref()
	if err != nil {
		return BLOB_UNKNOWN_SIZE, nil, err
	}
	defer a.Unref()

	size, acc, err := a.cache.GetBlobData(digest)
	if err != nil {
		if !blobaccess.IsErrBlobNotFound(err) {
			return BLOB_UNKNOWN_SIZE, nil, err
		}
		size, acc, err = a.source.GetBlobData(digest)
		if err == nil {
			acc = newCachedAccess(a, acc, size, digest)
		}
	}
	return size, acc, err
}

func (a *cached) AddBlob(blob blobaccess.BlobAccess) (int64, digest.Digest, error) {
	err := a.Ref()
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	defer a.Unref()

	if a.sink == nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, fmt.Errorf("no blob sink")
	}
	size, digest, err := a.cache.AddBlob(blob)
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	_, acc, err := a.cache.GetBlobData(digest)
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	size, digest, err = a.sink.AddBlob(blobaccess.ForDataAccess(digest, size, blob.MimeType(), acc))
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	return size, digest, err
}

func (c *cached) AddData(data blobaccess.DataAccess) (int64, digest.Digest, error) {
	return c.AddBlob(blobaccess.ForDataAccess(blobaccess.BLOB_UNKNOWN_DIGEST, BLOB_UNKNOWN_SIZE, "", data))
}

/////////////////////////////////////////

type cachedAccess struct {
	lock   sync.Mutex
	cache  *cached
	access blobaccess.DataAccess
	digest digest.Digest
	size   int64
	orig   blobaccess.DataAccess
}

var _ blobaccess.DataAccess = (*cachedAccess)(nil)

func CachedAccess(src BlobSource, dst BlobSink, cache BlobCache) (BlobCache, error) {
	var err error
	if cache == nil {
		cache, err = NewDefaultBlobCache()
		if err != nil {
			return nil, err
		}
	} else {
		err = cache.Ref()
		if err != nil {
			return nil, err
		}
	}
	if src != nil {
		err = src.Ref()
		if err != nil {
			return nil, err
		}
	}
	if dst != nil {
		err = dst.Ref()
		if err != nil {
			return nil, err
		}
	}
	c := &cached{source: src, sink: dst, cache: cache}
	c.Allocatable = refmgmt.NewAllocatable(c.cleanup)
	return c, nil
}

func newCachedAccess(cache *cached, blob blobaccess.DataAccess, size int64, digest digest.Digest) blobaccess.DataAccess {
	return &cachedAccess{
		cache:  cache,
		size:   size,
		digest: digest,
		orig:   blob,
	}
}

func (c *cachedAccess) Get() ([]byte, error) {
	var err error

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.access == nil && c.digest != "" {
		c.size, c.access, _ = c.cache.cache.GetBlobData(c.digest)
	}
	if c.access == nil {
		c.cache.lock.Lock()
		defer c.cache.lock.Unlock()

		if c.digest != "" {
			c.size, c.access, err = c.cache.cache.GetBlobData(c.digest)
			if err != nil && !blobaccess.IsErrBlobNotFound(err) {
				return nil, err
			}
		}
		if c.access == nil {
			data, err := c.orig.Get()
			if err != nil {
				return nil, err
			}
			c.size, c.digest, err = c.cache.cache.AddData(blobaccess.DataAccessForBytes(data))
			if err == nil {
				c.orig.Close()
				c.orig = nil
			}
			return data, err
		}
	}
	return c.access.Get()
}

func (c *cachedAccess) Reader() (io.ReadCloser, error) {
	var err error

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.access == nil && c.digest != "" {
		c.size, c.access, _ = c.cache.cache.GetBlobData(c.digest)
	}
	if c.access == nil {
		c.cache.lock.Lock()
		defer c.cache.lock.Unlock()

		if c.digest != "" {
			c.size, c.access, err = c.cache.cache.GetBlobData(c.digest)
			if err != nil && !blobaccess.IsErrBlobNotFound(err) {
				return nil, err
			}
		}
		if c.access == nil {
			c.size, c.digest, err = c.cache.cache.AddData(c.orig)
			if err == nil {
				_, c.access, err = c.cache.cache.GetBlobData(c.digest)
			}
			if err != nil {
				return nil, err
			}
			c.orig.Close()
			c.orig = nil
		}
	}
	return c.access.Reader()
}

func (c *cachedAccess) Close() error {
	return nil
}

func (c *cachedAccess) Size() int64 {
	return c.size
}

////////////////////////////////////////////////////////////////////////////////

type norefBlobSource struct {
	BlobSource
}

var _ BlobSource = (*norefBlobSource)(nil)

func NoRefBlobSource(s BlobSource) BlobSource { return &norefBlobSource{s} }

func (norefBlobSource) Ref() error {
	return nil
}

func (norefBlobSource) Unref() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type norefBlobSink struct {
	BlobSink
}

var _ BlobSink = (*norefBlobSink)(nil)

func NoRefBlobSink(s BlobSink) BlobSink { return &norefBlobSink{s} }

func (norefBlobSink) Ref() error {
	return nil
}

func (norefBlobSink) Unref() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type norefBlobCache struct {
	BlobCache
}

var _ BlobCache = (*norefBlobCache)(nil)

func NoRefBlobCache(s BlobCache) BlobCache { return &norefBlobCache{s} }

func (norefBlobCache) Ref() error {
	return nil
}

func (norefBlobCache) Unref() error {
	return nil
}
