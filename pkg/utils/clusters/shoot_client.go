// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package clusters

import (
	"context"
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver"
)

const subresourceAdminKubeconfig = "adminkubeconfig"

var shootGVR = schema.GroupVersionResource{
	Group:    "core.gardener.cloud",
	Version:  "v1beta1",
	Resource: "shoots",
}

type ShootClient struct {
	unstructuredClient dynamic.NamespaceableResourceInterface
}

func NewShootClient(gardenKubeconfigBytes []byte) (*ShootClient, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(gardenKubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("shoot client: unable to get rest config: %w", err)
	}

	cl, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("shoot client: unable to create dynamic client: %w", err)
	}

	return &ShootClient{
		unstructuredClient: cl.Resource(shootGVR),
	}, nil
}

func NewShootClientFromTarget(ctx context.Context, gardenTarget *v1alpha1.Target, targetResolver targetresolver.TargetResolver) (*ShootClient, error) {
	resolvedTarget, err := targetResolver.Resolve(ctx, gardenTarget)
	if err != nil {
		return nil, fmt.Errorf("shoot client: could not resolve target: %w", err)
	}

	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	err = json.Unmarshal([]byte(resolvedTarget.Content), targetConfig)
	if err != nil {
		return nil, fmt.Errorf("shoot client: failed to unmarshal target config: %w", err)
	}
	if targetConfig.Kubeconfig.StrVal == nil {
		return nil, fmt.Errorf("shoot client: target config contains no kubeconfig: %w", err)
	}

	gardenKubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)
	return NewShootClient(gardenKubeconfigBytes)
}

// ListShoots returns the list of shoot in the specified namespace
func (c *ShootClient) ListShoots(ctx context.Context, shootNamespace string) (*unstructured.UnstructuredList, error) {
	return c.unstructuredClient.Namespace(shootNamespace).List(ctx, metav1.ListOptions{})
}

func (c *ShootClient) GetShoot(ctx context.Context, shootNamespace, shootName string) (*unstructured.Unstructured, error) {
	return c.unstructuredClient.Namespace(shootNamespace).Get(ctx, shootName, metav1.GetOptions{})
}

func (c *ShootClient) ExistsShoot(ctx context.Context, shootNamespace, shootName string) (bool, error) {
	shoot, err := c.GetShoot(ctx, shootNamespace, shootName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	exists := shoot.GetDeletionTimestamp() == nil
	return exists, nil
}

// GetShootAdminKubeconfig returns a short-lived admin kubeconfig for the specified shoot as base64 encoded string.
func (c *ShootClient) GetShootAdminKubeconfig(ctx context.Context, shootName, shootNamespace string, kubeconfigExpirationSeconds int64) (string, metav1.Time, error) {

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

	var expirationTimestamp metav1.Time
	namespacedShootClient := c.unstructuredClient.Namespace(shootNamespace)
	result, err := namespacedShootClient.Create(ctx, &adminKubeconfigRequest, metav1.CreateOptions{}, subresourceAdminKubeconfig)
	if err != nil {
		return "", expirationTimestamp, fmt.Errorf("shoot client: admin kubeconfig request failed: %w", err)
	}

	shootKubeconfigBase64, found, err := unstructured.NestedString(result.Object, "status", "kubeconfig")
	if err != nil {
		return "", expirationTimestamp, fmt.Errorf("shoot client: could not get kubeconfig from result: %w", err)
	} else if !found {
		return "", expirationTimestamp, fmt.Errorf("shoot client: could not find kubeconfig in result")
	}
	rawExpirationTimestamp, found, err := unstructured.NestedString(result.Object, "status", "expirationTimestamp")
	if err != nil {
		return "", expirationTimestamp, fmt.Errorf("shoot client: could not get expiration timestamp from result: %w", err)
	} else if !found {
		return "", expirationTimestamp, fmt.Errorf("shoot client: could not find expiration timestamp in result")
	}
	err = expirationTimestamp.UnmarshalJSON([]byte(fmt.Sprintf("\"%s\"", rawExpirationTimestamp)))
	if err != nil {
		return "", expirationTimestamp, fmt.Errorf("error converting raw expiration timestamp '%s' to metav1.Time: %w", rawExpirationTimestamp, err)
	}

	return shootKubeconfigBase64, expirationTimestamp, nil
}
