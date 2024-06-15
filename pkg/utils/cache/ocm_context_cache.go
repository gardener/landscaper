package cache

import (
	"context"
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
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
	octx       ocm.Context
	registry   model.RegistryAccess
	blueprints map[client.ObjectKey]*blueprints.Blueprint
}

type BlueprintCacheID struct {
	jobID           string
	installationKey client.ObjectKey
}

func NewBlueprintCacheID(inst *v1alpha1.Installation) *BlueprintCacheID {
	return &BlueprintCacheID{
		jobID:           inst.Status.JobID,
		installationKey: client.ObjectKeyFromObject(inst),
	}
}

func (b *BlueprintCacheID) String() string {
	return "blueprintCacheID:" + b.jobID + "/" + b.installationKey.Namespace + "/" + b.installationKey.Name
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
			octx:       ocm.New(datacontext.MODE_EXTENDED),
			blueprints: make(map[client.ObjectKey]*blueprints.Blueprint),
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

func (o *OCMContextCache) GetBlueprint(ctx context.Context, id *BlueprintCacheID) *blueprints.Blueprint {
	if id == nil {
		return nil
	}

	o.rwLock.RLock()
	defer o.rwLock.RUnlock()

	entry := o.ocmEntries[id.jobID]
	if entry != nil {
		bp := entry.blueprints[id.installationKey]
		if bp != nil {
			log, _ := logging.FromContextOrNew(ctx, nil)
			log.Debug("get blueprint from ocm context cache", cacheKey, id.String())

			return bp
		}
	}

	return nil
}

func (o *OCMContextCache) AddBlueprint(ctx context.Context, blueprint *blueprints.Blueprint, id *BlueprintCacheID) {
	if id == nil {
		return
	}

	o.rwLock.Lock()
	defer o.rwLock.Unlock()

	entry := o.ocmEntries[id.jobID]
	if entry != nil {
		if entry.blueprints[id.installationKey] == nil {
			log, _ := logging.FromContextOrNew(ctx, nil)
			log.Debug("adding blueprint to ocm context cache", cacheKey, id.String())

			entry.blueprints[id.installationKey] = blueprint
		}
	}
}
