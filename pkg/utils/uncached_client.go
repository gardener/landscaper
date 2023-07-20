package utils

import (
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewUncachedClient(config *rest.Config, options client.Options) (client.Client, error) {
	options.Cache = nil

	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	return c, nil
}
