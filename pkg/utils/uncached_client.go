package utils

import (
	"fmt"
	"reflect"
	"strconv"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

func NewUncachedClient(config *rest.Config, options client.Options) (client.Client, error) {
	log, err := logging.GetLogger()

	if config.RateLimiter != nil {
		log.Info("NewUncachedClient-RateLimiter: " + reflect.TypeOf(config.RateLimiter).String())
	}

	log.Info("NewUncachedClient-OldBurst: " + strconv.Itoa(config.Burst))
	log.Info("NewUncachedClient-OldQPS: " + fmt.Sprintf("%v", config.QPS))

	config.RateLimiter = nil
	config.Burst = 1000
	config.QPS = 1000

	options.Cache = nil

	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	return c, nil
}
