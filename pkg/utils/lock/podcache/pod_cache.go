package podcache

import (
	"context"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

const PodCacheTimeout = 1 * time.Minute

func NewPodCache(hostClient client.Client) *PodCache {
	return &PodCache{
		pods:       map[string]bool{},
		hostClient: hostClient,
	}
}

type PodCache struct {
	hostClient client.Client
	pods       map[string]bool
	rwLock     sync.RWMutex
	lastUpdate time.Time
}

func (p *PodCache) PodExists(ctx context.Context, podKey client.ObjectKey) (bool, error) {
	exists := false

	isOutdated, exists := p.isOutdated(podKey)
	if isOutdated {
		exists, err := p.updateCache(ctx, podKey)
		return exists, err
	}

	if !exists {
		// we must not miss new pods and do their work in parallel
		tmpExists, err := podExist(ctx, p.hostClient, podKey)
		if err != nil {
			return false, err
		}

		exists = tmpExists

		if exists {
			p.addPod(podKey)
		}
	}

	return exists, nil
}

func (p *PodCache) isOutdated(podKey client.ObjectKey) (isOutdated bool, exists bool) {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	isOutdated = outdated(p.lastUpdate)
	exists = p.pods[podKey.String()]
	return isOutdated, exists
}

func (p *PodCache) updateCache(ctx context.Context, podKey client.ObjectKey) (bool, error) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()

	if outdated(p.lastUpdate) {
		newPods := utils.EmptyPodMetadataList()
		if err := read_write_layer.ListMetaData(ctx, p.hostClient, newPods, read_write_layer.R000081, client.InNamespace(podKey.Namespace)); err != nil {
			return false, err
		}

		p.pods = map[string]bool{}
		for i := range newPods.Items {
			nextKey := client.ObjectKeyFromObject(&newPods.Items[i])
			p.pods[nextKey.String()] = true
		}

		p.lastUpdate = time.Now()
	}

	exists := p.pods[podKey.String()]
	return exists, nil
}

func (p *PodCache) addPod(podKey client.ObjectKey) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	p.pods[podKey.String()] = true
}

// only for testing purposes
func (p *PodCache) TestSetLastUpdateTime(time time.Time) {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	p.lastUpdate = time
}

func outdated(lastUpdate time.Time) bool {
	return lastUpdate.Add(PodCacheTimeout).Before(time.Now())
}

func podExist(ctx context.Context, cl client.Client, key client.ObjectKey) (bool, error) {
	podMeta := utils.EmptyPodMetadata()

	if err := read_write_layer.GetMetaData(ctx, cl, key, podMeta, read_write_layer.R000082); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
