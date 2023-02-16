package token

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
)

const (
	subresourceAdminKubeconfig  = "adminkubeconfig"
	kubeconfigExpirationSeconds = 24 * 60 * 60
)

// GetShootAdminKubeconfigUsingGardenTarget returns a short-lived admin kubeconfig for the specified shoot as base64 encoded string.
func GetShootAdminKubeconfigUsingGardenTarget(ctx context.Context, target *v1alpha1.Target, shootName, shootNamespace string) (string, error) {
	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	err := json.Unmarshal(target.Spec.Configuration.RawMessage, targetConfig)
	if err != nil {
		return "", err
	}

	gardenKubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)

	return getShootAdminKubeconfigUsingGardenKubeconfig(ctx, gardenKubeconfigBytes, shootName, shootNamespace)
}

// getShootAdminKubeconfigUsingGardenKubeconfig returns a short-lived admin kubeconfig for the specified shoot as base64 encoded string.
func getShootAdminKubeconfigUsingGardenKubeconfig(ctx context.Context, gardenKubeconfigBytes []byte, shootName, shootNamespace string) (string, error) {
	shootClient, err := getShootClient(gardenKubeconfigBytes)
	if err != nil {
		return "", err
	}

	return getShootAdminKubeconfigUsingClient(ctx, shootClient, shootName, shootNamespace)
}

func getShootAdminKubeconfigUsingClient(ctx context.Context, shootClient dynamic.NamespaceableResourceInterface,
	shootName, shootNamespace string) (string, error) {

	adminKubeconfigRequest := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "authentication.gardener.cloud/v1alpha1",
			"kind":       "AdminKubeconfigRequest",
			"metadata": map[string]interface{}{
				"namespace": shootNamespace,
				"name":      shootName,
			},
			"spec": map[string]interface{}{
				"expirationSeconds": kubeconfigExpirationSeconds,
			},
		},
	}

	namespacedShootClient := shootClient.Namespace(shootNamespace)
	result, err := namespacedShootClient.Create(ctx, &adminKubeconfigRequest, metav1.CreateOptions{}, subresourceAdminKubeconfig)
	if err != nil {
		return "", fmt.Errorf("admin kubeconfig request failed: %w", err)
	}

	shootKubeconfigBase64, found, err := unstructured.NestedString(result.Object, "status", "kubeconfig")
	if err != nil {
		return "", fmt.Errorf("admin kubeconfig request: could not get kubeconfig from result: %w", err)
	} else if !found {
		return "", fmt.Errorf("admin kubeconfig request: could not find kubeconfig in result")
	}

	return shootKubeconfigBase64, nil
}
