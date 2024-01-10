package utils

import (
	"fmt"
	"reflect"
	"strconv"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

func ClientsFromManagers(lsMgr, hostMgr manager.Manager) (
	lsUncachedClient,
	lsCachedClient,
	hostUncachedClient,
	hostCachedClient client.Client,
	err error,
) {
	lsUncachedClient, err = NewUncachedClientFromManager(lsMgr)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("unable to build new uncached ls client: %w", err)
	}

	lsCachedClient = lsMgr.GetClient()

	hostUncachedClient, err = NewUncachedClientFromManager(hostMgr)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("unable to build new uncached host client: %w", err)
	}

	hostCachedClient = hostMgr.GetClient()

	return lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient, nil
}

func NewUncachedClientFromManager(mgr manager.Manager) (client.Client, error) {
	return client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
}

func NewUncachedClient(burst, qps int) func(config *rest.Config, options client.Options) (client.Client, error) {

	return func(config *rest.Config, options client.Options) (client.Client, error) {
		options.Cache = nil

		log, err := logging.GetLogger()
		if err != nil {
			return nil, err
		}

		configCopy := *config

		if config.RateLimiter != nil {
			log.Info("NewUncachedClient-RateLimiter: " + reflect.TypeOf(config.RateLimiter).String())
		}
		log.Info("NewUncachedClient-OldBurst: " + strconv.Itoa(config.Burst))
		log.Info("NewUncachedClient-OldQPS: " + fmt.Sprintf("%v", config.QPS))

		configCopy.RateLimiter = nil
		configCopy.Burst = burst
		configCopy.QPS = float32(qps)

		c, err := client.New(&configCopy, options)
		if err != nil {
			return nil, err
		}

		return c, nil
	}

}

func NewUncached(burst, qps int, config *rest.Config, options client.Options) (client.Client, error) {
	return NewUncachedClient(burst, qps)(config, options)
}

func NewForConfig(burst, qps int, config *rest.Config) (*kubernetes.Clientset, error) {
	var configCopy *rest.Config
	if config != nil {
		tmp := *config
		configCopy = &tmp

		log, err := logging.GetLogger()
		if err != nil {
			return nil, err
		}

		if config.RateLimiter != nil {
			log.Info("NewForConfig-RateLimiter: " + reflect.TypeOf(config.RateLimiter).String())
		}
		log.Info("NewForConfig-OldBurst: " + strconv.Itoa(config.Burst))
		log.Info("NewForConfig-OldQPS: " + fmt.Sprintf("%v", config.QPS))

		configCopy.RateLimiter = nil
		configCopy.Burst = burst
		configCopy.QPS = float32(qps)

	}

	return kubernetes.NewForConfig(configCopy)
}
