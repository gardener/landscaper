package chartresolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/gardener/landscaper/pkg/utils/resource_cache"

	"helm.sh/helm/v3/pkg/chart"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

type HelmChartCache struct {
	maxSizeInByte          int64
	removeOutdatedDuration time.Duration
}

func GetHelmChartCache(initMaxSizeInByte int64, initRemoveOutdatedDuration time.Duration) *HelmChartCache {
	return &HelmChartCache{maxSizeInByte: initMaxSizeInByte, removeOutdatedDuration: initRemoveOutdatedDuration}
}

func (c *HelmChartCache) getChart(ociRef string, helmRepo *helmv1alpha1.HelmChartRepo, ocmKey string) (*chart.Chart, error) {
	hash, err := c.getHash(ociRef, helmRepo, ocmKey)

	if err != nil {
		return nil, err
	}

	helmChart := &chart.Chart{}
	if err = resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).GetEntry(hash, helmChart); err != nil {
		return nil, err
	}

	return helmChart, nil
}

func (c *HelmChartCache) HasKey(ociRef string, helmRepo *helmv1alpha1.HelmChartRepo, ocmKey string) (bool, error) {
	hash, err := c.getHash(ociRef, helmRepo, ocmKey)

	if err != nil {
		return false, err
	}

	return resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).HasKey(hash)
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

	return resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).AddOrUpdate(ctx, hash, chart)
}

func (c *HelmChartCache) Clear() {
	resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).Clear()
}

func (c *HelmChartCache) GetEntries() (map[string]*resource_cache.CacheEntry, int64, time.Time) {
	return resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).GetEntries()
}

func (c *HelmChartCache) SetMaxSizeInByte(maxSizeInByte int64) {
	resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).SetMaxSizeInByte(maxSizeInByte)
}

func (c *HelmChartCache) SetOutdatedDuration(removeOutdatedDuration time.Duration) {
	resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).SetOutdatedDuration(removeOutdatedDuration)
}

func (c *HelmChartCache) SetLastCleanup(lastCleanup time.Time) {
	resource_cache.GetResourceCache(c.maxSizeInByte, c.removeOutdatedDuration).SetLastCleanup(lastCleanup)
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
