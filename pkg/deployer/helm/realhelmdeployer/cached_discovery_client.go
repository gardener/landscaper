package realhelmdeployer

import (
	"k8s.io/client-go/discovery"
)

// cachedDiscoveryClient
// implements the interface CachedDiscoveryInterface without the cache functionality. We need this interface to interact
// with the k8s interface RESTClientGetter (see method ToDiscoveryClient in remoteRESTClientGetter).
type cachedDiscoveryClient struct {
	discovery.DiscoveryInterface
}

func (cachedDiscoveryClient) Fresh() bool {
	return true
}

func (cachedDiscoveryClient) Invalidate() {
}
