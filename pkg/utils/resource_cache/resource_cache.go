package resource_cache

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/gardener/landscaper/apis/utils"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// TODO
// make stuff configurable

const (
	MaxSizeInByteDefault          = 100 * 1000 * 1000
	RemoveOutdatedDurationDefault = time.Hour * 24
)

type ResourceCache struct {
	resourceCache     map[string]*CacheEntry
	rwLock            sync.RWMutex
	currentSizeInByte int64
	lastCleanup       time.Time

	maxSizeInByte          int64
	removeOutdatedDuration time.Duration
}

type CacheEntry struct {
	resourceBytesCompressed []byte
	timestamp               time.Time
}

func (c *CacheEntry) GetEntries() ([]byte, time.Time) {
	return c.resourceBytesCompressed, c.timestamp
}

var resourceCache *ResourceCache
var getInstanceLock sync.RWMutex

func GetResourceCache(initMaxSizeInByte int64, initRemoveOutdatedDuration time.Duration) *ResourceCache {
	cache := getResourceCacheSync()
	if cache != nil {
		return cache
	}

	cache = createResourceCacheSync(initMaxSizeInByte, initRemoveOutdatedDuration)
	return cache
}

func getResourceCacheSync() *ResourceCache {
	getInstanceLock.RLock()
	defer getInstanceLock.RUnlock()
	return resourceCache
}

func createResourceCacheSync(initMaxSizeInByte int64, initRemoveOutdatedDuration time.Duration) *ResourceCache {
	getInstanceLock.Lock()
	defer getInstanceLock.Unlock()

	if resourceCache == nil {
		resourceCache = &ResourceCache{
			resourceCache:          make(map[string]*CacheEntry),
			currentSizeInByte:      0,
			lastCleanup:            time.Now(),
			maxSizeInByte:          initMaxSizeInByte,
			removeOutdatedDuration: initRemoveOutdatedDuration,
		}
	}

	return resourceCache
}

func (c *ResourceCache) GetEntry(hash string, obj any) error {
	resourceBytesCompressed := c.getResourceBytesCompressed(hash)
	if len(resourceBytesCompressed) == 0 {
		return nil
	}

	resourceUncompressed, err := utils.Gunzip(resourceBytesCompressed)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(resourceUncompressed, obj); err != nil {
		return err
	}

	return nil
}

func (c *ResourceCache) getResourceBytesCompressed(hash string) []byte {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	entry := c.resourceCache[hash]

	if entry == nil {
		return nil
	}

	return entry.resourceBytesCompressed
}

func (c *ResourceCache) HasKey(hash string) (bool, error) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	entry := c.resourceCache[hash]
	if entry == nil {
		return false, nil
	}

	return true, nil
}

func (c *ResourceCache) AddOrUpdate(ctx context.Context, hash string, obj any) error {
	if obj == nil {
		return errors.New("object is nil")
	}

	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	entry := c.resourceCache[hash]
	if entry == nil {
		entry, err := c.createEntry(obj)
		if err != nil {
			return err
		}

		c.resourceCache[hash] = entry
		c.currentSizeInByte += int64(len(entry.resourceBytesCompressed))
	} else {
		entry.timestamp = time.Now()
	}

	for c.currentSizeInByte > c.maxSizeInByte {
		c.removeOldest()
	}

	if c.lastCleanup.Add(time.Hour).Before(time.Now()) {
		c.removeOutdated(ctx)
		c.lastCleanup = time.Now()
	}

	return nil
}

func (c *ResourceCache) Clear() {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.resourceCache = make(map[string]*CacheEntry)
	c.currentSizeInByte = 0
	c.lastCleanup = time.Now()
}

func (c *ResourceCache) GetEntries() (map[string]*CacheEntry, int64, time.Time) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	return c.resourceCache, c.currentSizeInByte, c.lastCleanup
}

func (c *ResourceCache) SetMaxSizeInByte(maxSizeInByte int64) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.maxSizeInByte = maxSizeInByte
}

func (c *ResourceCache) SetOutdatedDuration(removeOutdatedDuration time.Duration) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.removeOutdatedDuration = removeOutdatedDuration
}

func (c *ResourceCache) SetLastCleanup(lastCleanup time.Time) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.lastCleanup = lastCleanup
}

func (c *ResourceCache) createEntry(obj any) (*CacheEntry, error) {
	var resourceMarshaled []byte
	resourceMarshaled, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	resourceCompressed, err := utils.Gzip(resourceMarshaled)
	if err != nil {
		return nil, err
	}

	return &CacheEntry{
		resourceBytesCompressed: resourceCompressed,
		timestamp:               time.Now(),
	}, nil
}

func (c *ResourceCache) removeOldest() {
	var oldestEntry *CacheEntry
	oldestHash := ""
	for hash, entry := range c.resourceCache {
		if oldestEntry == nil || entry.timestamp.Before(oldestEntry.timestamp) {
			oldestHash = hash
			oldestEntry = entry
		}
	}

	if oldestEntry != nil {
		c.deleteEntry(oldestHash)
	}
}

func (c *ResourceCache) removeOutdated(ctx context.Context) {
	logger, _ := logging.FromContextOrNew(ctx, nil)

	hashes := []string{}
	for hash, entry := range c.resourceCache {
		if entry.timestamp.Add(c.removeOutdatedDuration).Before(time.Now()) {
			hashes = append(hashes, hash)
		}
	}

	for _, hash := range hashes {
		c.deleteEntry(hash)
	}

	logger.Info("ResourceCacheStatistics: Elements: " + strconv.Itoa(len(c.resourceCache)) +
		"/Size: " + strconv.FormatInt(c.currentSizeInByte, 10))

}

func (c *ResourceCache) deleteEntry(hash string) {
	c.currentSizeInByte -= int64(len(c.resourceCache[hash].resourceBytesCompressed))
	delete(c.resourceCache, hash)
}
