package utils

import (
	"fmt"
	"reflect"
	"strconv"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

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
