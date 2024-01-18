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
	nameResourceVersion, ok := c.namespaceObjects[m.Namespace]

	if !ok {
		nameResourceVersion = map[string]string{}
		c.namespaceObjects[m.Namespace] = nameResourceVersion
	}

	nameResourceVersion[m.Name] = m.ResourceVersion
}

func (c *FinishedObjectCache) IsFinishedAndRemove(m *metav1.PartialObjectMetadata) bool {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	nameResourceVersion, ok := c.namespaceObjects[m.Namespace]

	if !ok {
		return false
	}

	resourceVersion, ok := nameResourceVersion[m.Name]
	if !ok {
		return false
	}

	delete(nameResourceVersion, m.Name)

	if len(c.namespaceObjects[m.Namespace]) == 0 {
		delete(c.namespaceObjects, m.Namespace)
	}

	if m.ResourceVersion != resourceVersion {
		return false
	}

	return true
}

func (c *FinishedObjectCache) IsContained(req reconcile.Request) bool {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	nameResourceVersion, ok := c.namespaceObjects[req.Namespace]

	if !ok {
		return false
	}

	_, ok = nameResourceVersion[req.Name]
	return ok
}
