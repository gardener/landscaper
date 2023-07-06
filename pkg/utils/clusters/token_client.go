// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package clusters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/pkg/utils/targetresolver"
)

type TokenClient struct {
	kubeconfig []byte
	clientset  *kubernetes.Clientset
}

func NewTokenClient(kubeconfigBytes []byte) (*TokenClient, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("token client: unable to get rest config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("token client: unable to create clientset: %w", err)
	}

	return &TokenClient{
		kubeconfig: kubeconfigBytes,
		clientset:  clientset,
	}, nil
}

func NewTokenClientFromTarget(ctx context.Context, target *v1alpha1.Target, targetResolver targetresolver.TargetResolver) (*TokenClient, error) {
	resolvedTarget, err := targetResolver.Resolve(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("token client: could not resolve target: %w", err)
	}

	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	err = json.Unmarshal([]byte(resolvedTarget.Content), targetConfig)
	if err != nil {
		return nil, fmt.Errorf("token client: failed to unmarshal target config: %w", err)
	}
	if targetConfig.Kubeconfig.StrVal == nil {
		return nil, fmt.Errorf("token client: target config contains no kubeconfig: %w", err)
	}

	kubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)
	return NewTokenClient(kubeconfigBytes)
}

func (c *TokenClient) GetServiceAccountToken(ctx context.Context, serviceAccountName, serviceAccountNamespace string,
	expirationSeconds int64) (string, error) {

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: pointer.Int64(expirationSeconds),
		},
	}

	serviceAccountClient := c.clientset.CoreV1().ServiceAccounts(serviceAccountNamespace)
	tokenRequest, err := serviceAccountClient.CreateToken(ctx, serviceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("token client: token request failed: %w", err)
	}

	return tokenRequest.Status.Token, nil
}

func (c *TokenClient) GetServiceAccountKubeconfig(ctx context.Context, serviceAccountName, serviceAccountNamespace string,
	expirationSeconds int64) (string, error) {

	token, err := c.GetServiceAccountToken(ctx, serviceAccountName, serviceAccountNamespace, expirationSeconds)
	if err != nil {
		return "", err
	}

	config, err := clientcmd.Load(c.kubeconfig)
	if err != nil {
		return "", fmt.Errorf("token client: failed to load config: %w", err)
	}

	context, ok := config.Contexts[config.CurrentContext]
	if !ok || context == nil {
		return "", fmt.Errorf("token client: current context not found: %w", err)
	}

	context.AuthInfo = serviceAccountName
	config.Contexts = map[string]*api.Context{
		config.CurrentContext: context,
	}

	config.AuthInfos = map[string]*api.AuthInfo{
		serviceAccountName: &api.AuthInfo{
			Token: token,
		},
	}

	cluster, ok := config.Clusters[context.Cluster]
	if !ok || context == nil {
		return "", fmt.Errorf("token client: current cluster not found: %w", err)
	}

	config.Clusters = map[string]*api.Cluster{
		context.Cluster: cluster,
	}

	serviceAccountKubeconfig, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("token client: failed to write config: %w", err)
	}

	serviceAccountKubeconfig64 := base64.StdEncoding.EncodeToString(serviceAccountKubeconfig)

	return serviceAccountKubeconfig64, nil
}
