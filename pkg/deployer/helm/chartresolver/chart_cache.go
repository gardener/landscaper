package chartresolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/gardener/landscaper/apis/utils"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"helm.sh/helm/v3/pkg/chart"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

// TODO
// unit tests
// make stuff configurable

const (
	MaxSizeInByteDefault          = 100 * 1000 * 1000
	RemoveOutdatedDurationDefault = time.Hour * 24
)

type HelmChartCache struct {
	chartCache        map[string]*cacheEntry
	rwLock            sync.RWMutex
	currentSizeInByte int64
	lastCleanup       time.Time

	maxSizeInByte          int64
	removeOutdatedDuration time.Duration
}

type cacheEntry struct {
	chartBytesCompressed []byte
	timestamp            time.Time
}

func (c *cacheEntry) GetEntries() ([]byte, time.Time) {
	return c.chartBytesCompressed, c.timestamp
}

var chartCache *HelmChartCache
var getInstanceLock sync.RWMutex

func GetHelmChartCache(initMaxSizeInByte int64, initRemoveOutdatedDuration time.Duration) *HelmChartCache {
	cache := getHelmChartCacheSync()
	if cache != nil {
		return cache
	}

	cache = createHelmChartCacheSync(initMaxSizeInByte, initRemoveOutdatedDuration)
	return cache
}

func getHelmChartCacheSync() *HelmChartCache {
	getInstanceLock.RLock()
	defer getInstanceLock.RUnlock()
	return chartCache
}

func createHelmChartCacheSync(initMaxSizeInByte int64, initRemoveOutdatedDuration time.Duration) *HelmChartCache {
	getInstanceLock.Lock()
	defer getInstanceLock.Unlock()

	if chartCache == nil {
		chartCache = &HelmChartCache{
			chartCache:             make(map[string]*cacheEntry),
			currentSizeInByte:      0,
			lastCleanup:            time.Now(),
			maxSizeInByte:          initMaxSizeInByte,
			removeOutdatedDuration: initRemoveOutdatedDuration,
		}
	}

	return chartCache
}

func (c *HelmChartCache) getChart(ociRef string, helmRepo *helmv1alpha1.HelmChartRepo, ocmKey string) (*chart.Chart, error) {
	hash, err := c.getHash(ociRef, helmRepo, ocmKey)

	if err != nil {
		return nil, err
	}

	chartBytesCompressed := c.getChartBytesCompressed(hash)
	if len(chartBytesCompressed) == 0 {
		return nil, nil
	}

	chartUncompressed, err := utils.Gunzip(chartBytesCompressed)
	if err != nil {
		return nil, err
	}

	helmChart := &chart.Chart{}
	if err := json.Unmarshal(chartUncompressed, helmChart); err != nil {
		return nil, err
	}

	return helmChart, nil
}

func (c *HelmChartCache) getChartBytesCompressed(hash string) []byte {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	entry := c.chartCache[hash]

	if entry == nil {
		return nil
	}

	return entry.chartBytesCompressed
}

func (c *HelmChartCache) HasKey(ociRef string, helmRepo *helmv1alpha1.HelmChartRepo, ocmKey string) (bool, error) {
	hash, err := c.getHash(ociRef, helmRepo, ocmKey)

	if err != nil {
		return false, err
	}

	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	entry := c.chartCache[hash]
	if entry == nil {
		return false, nil
	}

	return true, nil
}

func (c *HelmChartCache) addOrUpdateChart(ctx context.Context, ociRef string, helmRepo *helmv1alpha1.HelmChartRepo,
	ocmKey string, chart *chart.Chart) error {

	if chart == nil {
		return errors.New("chart is nil")
	}

	hash, err := c.getHash(ociRef, helmRepo, ocmKey)
	if err != nil {
		return err
	}

	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	entry := c.chartCache[hash]
	if entry == nil {
		entry, err = c.createEntry(chart)
		if err != nil {
			return err
		}

		c.chartCache[hash] = entry
		c.currentSizeInByte += int64(len(entry.chartBytesCompressed))
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

func (c *HelmChartCache) Clear() {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.chartCache = make(map[string]*cacheEntry)
	c.currentSizeInByte = 0
	c.lastCleanup = time.Now()
}

func (c *HelmChartCache) GetEntries() (map[string]*cacheEntry, int64, time.Time) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	return c.chartCache, c.currentSizeInByte, c.lastCleanup
}

func (c *HelmChartCache) SetMaxSizeInByte(maxSizeInByte int64) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.maxSizeInByte = maxSizeInByte
}

func (c *HelmChartCache) SetOutdatedDuration(removeOutdatedDuration time.Duration) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.removeOutdatedDuration = removeOutdatedDuration
}

func (c *HelmChartCache) SetLastCleanup(lastCleanup time.Time) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.lastCleanup = lastCleanup
}

func (c *HelmChartCache) createEntry(chart *chart.Chart) (*cacheEntry, error) {
	var chartMarshaled []byte
	chartMarshaled, err := json.Marshal(chart)
	if err != nil {
		return nil, err
	}

	chartCompressed, err := utils.Gzip(chartMarshaled)
	if err != nil {
		return nil, err
	}

	return &cacheEntry{
		chartBytesCompressed: chartCompressed,
		timestamp:            time.Now(),
	}, nil
}

func (c *HelmChartCache) getHash(ociRef string, helmRepo *helmv1alpha1.HelmChartRepo, ocmKey string) (string, error) {
	var bytes []byte
	var err error

	if len(ociRef) != 0 {
		bytes, err = json.Marshal(ociRef)
	} else if helmRepo != nil {
		bytes, err = json.Marshal(*helmRepo)
	} else if ocmKey != "" {
		bytes, err = json.Marshal(ocmKey)
	} else {
		return "", NoChartDefinedError
	}

	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(bytes)
	hashStr := hex.EncodeToString(hash[:])

	return hashStr, nil
}

func (c *HelmChartCache) removeOldest() {
	var oldestEntry *cacheEntry
	oldestHash := ""
	for hash, entry := range c.chartCache {
		if oldestEntry == nil || entry.timestamp.Before(oldestEntry.timestamp) {
			oldestHash = hash
			oldestEntry = entry
		}
	}

	if oldestEntry != nil {
		c.deleteEntry(oldestHash)
	}
}

func (c *HelmChartCache) removeOutdated(ctx context.Context) {
	logger, _ := logging.FromContextOrNew(ctx, nil)

	hashes := []string{}
	for hash, entry := range c.chartCache {
		if entry.timestamp.Add(c.removeOutdatedDuration).Before(time.Now()) {
			hashes = append(hashes, hash)
		}
	}

	for _, hash := range hashes {
		c.deleteEntry(hash)
	}

	logger.Info("HelmChartCacheStatistics: Elements: " + strconv.Itoa(len(c.chartCache)) +
		"/Size: " + strconv.FormatInt(c.currentSizeInByte, 10))

}

func (c *HelmChartCache) deleteEntry(hash string) {
	c.currentSizeInByte -= int64(len(c.chartCache[hash].chartBytesCompressed))
	delete(c.chartCache, hash)
}
