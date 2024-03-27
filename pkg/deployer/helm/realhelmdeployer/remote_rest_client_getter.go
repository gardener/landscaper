// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type remoteRESTClientGetter struct {
	config    *rest.Config
	namespace string
}

func newRemoteRESTClientGetter(config *rest.Config, namespace string) *remoteRESTClientGetter {
	return &remoteRESTClientGetter{
		config:    config,
		namespace: namespace,
	}
}

func (k *remoteRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return &ClientConfigGetter{
		config:    k.config,
		namespace: k.namespace,
	}
}

// ToRESTConfig returns restconfig
func (k *remoteRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return k.ToRawKubeConfigLoader().ClientConfig()
}

// ToDiscoveryClient returns discovery client
func (k *remoteRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	restConfig, err := k.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	client := cachedDiscoveryClient{discoveryClient}

	return client, err
}

// ToRESTMapper returns a restmapper
func (k *remoteRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := k.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient, nil)
	return expander, nil
}
