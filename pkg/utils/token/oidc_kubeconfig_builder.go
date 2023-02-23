package token

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/pkg/utils/targetresolver"
)

func BuildOIDCKubeconfig(ctx context.Context, issuerURL, clientID string, target *v1alpha1.Target,
	targetResolver targetresolver.TargetResolver) (string, error) {

	resolvedTarget, err := targetResolver.Resolve(ctx, target)
	if err != nil {
		return "", fmt.Errorf("oidc kubeconfig builder: could not resolve target: %w", err)
	}

	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	err = json.Unmarshal([]byte(resolvedTarget.Content), targetConfig)
	if err != nil {
		return "", fmt.Errorf("oidc kubeconfig builder: failed to unmarshal target config: %w", err)
	}
	if targetConfig.Kubeconfig.StrVal == nil {
		return "", fmt.Errorf("oidc kubeconfig builder: target config contains no kubeconfig: %w", err)
	}

	kubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)

	config, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return "", fmt.Errorf("oidc kubeconfig builder: failed to load config: %w", err)
	}

	context, ok := config.Contexts[config.CurrentContext]
	if !ok || context == nil {
		return "", fmt.Errorf("oidc kubeconfig builder: current context not found: %w", err)
	}

	config.Contexts = map[string]*api.Context{
		config.CurrentContext: context,
	}

	cluster, ok := config.Clusters[context.Cluster]
	if !ok || context == nil {
		return "", fmt.Errorf("oidc kubeconfig builder: current cluster not found: %w", err)
	}

	config.Clusters = map[string]*api.Cluster{
		context.Cluster: cluster,
	}

	config.AuthInfos = map[string]*api.AuthInfo{
		context.AuthInfo: {
			Exec: &api.ExecConfig{
				APIVersion: "client.authentication.k8s.io/v1beta1",
				Command:    "kubectl",
				Args: []string{
					"oidc-login",
					"get-token",
					fmt.Sprintf("--oidc-issuer-url=%s", issuerURL),
					fmt.Sprintf("--oidc-client-id=%s", clientID),
					"--oidc-extra-scope=email",
					"--oidc-extra-scope=profile",
					"--oidc-extra-scope=offline_access",
					"--oidc-use-pkce",
					"--grant-type=auto",
				},
			},
		},
	}

	oidcKubeconfig, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("oidc kubeconfig builder: failed to write config: %w", err)
	}

	oidcKubeconfig64 := base64.StdEncoding.EncodeToString(oidcKubeconfig)

	return oidcKubeconfig64, nil
}
