package token

import (
	"context"
	"encoding/json"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/pkg/utils/targetresolver"
)

type TokenClient struct {
	clientset *kubernetes.Clientset
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
		clientset: clientset,
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

	tokenRequest, err := c.clientset.CoreV1().ServiceAccounts(serviceAccountNamespace).CreateToken(ctx,
		serviceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("token client: failed to fetch token: %w", err)
	}

	return tokenRequest.Status.Token, nil
}
