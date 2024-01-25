package utils

import (
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FinishedObjectCache struct {
	namespaceObjects map[string]nameResourceVersion
	rwLock           sync.RWMutex
}

type nameResourceVersion = map[string]string

func NewFinishedObjectCache() *FinishedObjectCache {
	return &FinishedObjectCache{
		namespaceObjects: map[string]nameResourceVersion{},
	}
}

func (c *FinishedObjectCache) Add(m *metav1.ObjectMeta) {

	if !m.DeletionTimestamp.IsZero() {
		return
	}

	namespaceResourceVersions, ok := c.namespaceObjects[m.Namespace]

	if !ok {
		namespaceResourceVersions = map[string]string{}
		c.namespaceObjects[m.Namespace] = namespaceResourceVersions
	}

	namespaceResourceVersions[m.Name] = m.ResourceVersion
}

func (c *FinishedObjectCache) AddSynchonized(m *metav1.ObjectMeta) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	c.Add(m)
}

func (c *FinishedObjectCache) IsFinishedOrRemove(m *metav1.PartialObjectMetadata) bool {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	namespaceResourceVersions, ok := c.namespaceObjects[m.Namespace]

	if !ok {
		return false
	}

	resourceVersion, ok := namespaceResourceVersions[m.Name]
	if !ok {
		return false
	}

	if m.ResourceVersion != resourceVersion {
		delete(namespaceResourceVersions, m.Name)

		if len(namespaceResourceVersions) == 0 {
			delete(c.namespaceObjects, m.Namespace)
		}

		return false
	}

	return true
}

func (c *FinishedObjectCache) IsContained(req reconcile.Request) bool {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	namespaceResourceVersions, ok := c.namespaceObjects[req.Namespace]

	if !ok {
		return false
	}

	_, ok = namespaceResourceVersions[req.Name]
	return ok
}
