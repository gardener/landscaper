// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package repocpi

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/errors"
)

type (
	BlobCacheEntry = blobaccess.BlobAccess
	BlobCacheKey   = interface{}
)

type BlobCache interface {
	// AddBlobFor stores blobs for added blobs not yet accessible
	// by generated access method until version is finally added.
	AddBlobFor(acc BlobCacheKey, blob BlobCacheEntry) error

	// GetBlobFor retrieves the original blob access for
	// a given access specification.
	GetBlobFor(acc BlobCacheKey) BlobCacheEntry

	RemoveBlobFor(acc BlobCacheKey)
	Clear() error
}

type blobCache struct {
	lock      sync.Mutex
	blobcache map[BlobCacheKey]BlobCacheEntry
}

func NewBlobCache() BlobCache {
	return &blobCache{
		blobcache: map[BlobCacheKey]BlobCacheEntry{},
	}
}

func (c *blobCache) RemoveBlobFor(acc BlobCacheKey) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if b := c.blobcache[acc]; b != nil {
		b.Close()
		delete(c.blobcache, acc)
	}
}

func (c *blobCache) AddBlobFor(acc BlobCacheKey, blob BlobCacheEntry) error {
	if s, ok := acc.(string); ok && s == "" {
		return errors.ErrInvalid("blob key")
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.blobcache[acc] == nil {
		l, err := blob.Dup()
		if err != nil {
			return err
		}
		c.blobcache[acc] = l
	}
	return nil
}

func (c *blobCache) GetBlobFor(acc BlobCacheKey) BlobCacheEntry {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.blobcache[acc]
}

func (c *blobCache) Clear() error {
	list := errors.ErrList()
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, b := range c.blobcache {
		list.Add(b.Close())
	}
	c.blobcache = map[BlobCacheKey]BlobCacheEntry{}
	return list.Result()
}
