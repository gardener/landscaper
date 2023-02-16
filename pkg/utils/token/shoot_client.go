package token

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	shootGVR = schema.GroupVersionResource{
		Group:    "core.gardener.cloud",
		Version:  "v1beta1",
		Resource: "shoots",
	}
)

func getShootClient(gardenKubeconfigBytes []byte) (dynamic.NamespaceableResourceInterface, error) {

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(gardenKubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to get rest config: %w", err)
	}

	cl, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create shoot client: %w", err)
	}

	return cl.Resource(shootGVR), nil
}
