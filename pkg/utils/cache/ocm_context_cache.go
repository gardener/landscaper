package cache

import (
	"context"
	"sync"

	"github.com/gardener/landscaper/pkg/components/model"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

const cacheKey = "cacheKey"

func init() {
	ocmContextCacheInstance = &OCMContextCache{
		ocmEntries: make(map[string]*OCMContextCacheEntry),
	}
}

type OCMContextCache struct {
	ocmEntries map[string]*OCMContextCacheEntry
	rwLock     sync.RWMutex
}

type OCMContextCacheEntry struct {
	octx     ocm.Context
	registry model.RegistryAccess
}

var ocmContextCacheInstance *OCMContextCache

func GetOCMContextCache() *OCMContextCache {
	return ocmContextCacheInstance
}

func (o *OCMContextCache) GetOrCreateOCMContext(ctx context.Context, jobID string) ocm.Context {
	entry := o.getOCMCacheEntry(jobID)
	if entry != nil {
		return entry.octx
	}

	o.rwLock.Lock()
	defer o.rwLock.Unlock()

	if o.ocmEntries[jobID] == nil {
		log, _ := logging.FromContextOrNew(ctx, nil)
		log.Debug("creating new ocm context", cacheKey, jobID)
		o.ocmEntries[jobID] = &OCMContextCacheEntry{
			octx: ocm.New(datacontext.MODE_EXTENDED),
		}
	}

	return o.ocmEntries[jobID].octx
}

func (o *OCMContextCache) getOCMCacheEntry(jobID string) *OCMContextCacheEntry {
	o.rwLock.RLock()
	defer o.rwLock.RUnlock()

	return o.ocmEntries[jobID]
}

func (o *OCMContextCache) RemoveOCMContext(ctx context.Context, jobID string) error {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()

	entry := o.ocmEntries[jobID]
	if entry != nil {
		log, _ := logging.FromContextOrNew(ctx, nil)
		log.Debug("removing ocm context from cache", cacheKey, jobID)

		delete(o.ocmEntries, jobID)

		if err := entry.octx.Finalize(); err != nil {
			log.Error(err, "failed to finalize ocm context", cacheKey, jobID)
			return err
		}
	}

	return nil
}

func (o *OCMContextCache) GetRegistryAccess(ctx context.Context, jobID string) model.RegistryAccess {
	entry := o.getOCMCacheEntry(jobID)
	if entry != nil {
		log, _ := logging.FromContextOrNew(ctx, nil)
		log.Debug("get registry from ocm context cache", cacheKey, jobID)

		return entry.registry
	}

	return nil
}

func (o *OCMContextCache) AddRegistryAccess(ctx context.Context, jobID string, registry model.RegistryAccess) {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()

	entry := o.ocmEntries[jobID]
	if entry != nil {
		log, _ := logging.FromContextOrNew(ctx, nil)
		log.Debug("adding registry to ocm context cache", cacheKey, jobID)

		entry.registry = registry
	}
}
