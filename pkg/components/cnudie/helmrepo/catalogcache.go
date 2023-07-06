package helmrepo

import (
	"crypto/sha256"
	"fmt"

	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

type catalogCacheEntry struct {
	checksum string
	index    *repo.IndexFile
}

type CatalogCache map[string]*catalogCacheEntry

// nolint
var catalogCache = CatalogCache{}

func getCatalogCache() *CatalogCache {
	return &catalogCache
}

// Cache the result of parsing the repo index since parsing this YAML
// is an expensive operation. See https://github.wdf.sap.corp/kubernetes/hub/issues/1052
func (c *CatalogCache) getCatalogFromCache(repoURL string, data []byte) (*repo.IndexFile, string) {
	sha := c.checksum(data)
	if catalogCache[repoURL] == nil || catalogCache[repoURL].checksum != sha {
		// The repository is not in the cache or the content changed
		return nil, sha
	}
	return catalogCache[repoURL].index, sha
}

func (c *CatalogCache) checksum(data []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write(data)
	return string(hasher.Sum(nil))
}

func (c *CatalogCache) parseCatalog(data []byte) (*repo.IndexFile, error) {
	index := &repo.IndexFile{}
	err := yaml.Unmarshal(data, index)
	if err != nil {
		return index, fmt.Errorf("could not unmarshall helm chart repo index: %w", err)
	}
	index.SortEntries()
	return index, nil
}

func (c *CatalogCache) storeCatalogInCache(repoURL string, index *repo.IndexFile, sha string) {
	catalogCache[repoURL] = &catalogCacheEntry{sha, index}
}
