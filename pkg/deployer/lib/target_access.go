package lib

import (
	"context"
	"errors"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
)

// defaultExpirationSeconds is the default expiration duration for tokens generated for OIDC Targets and Self Targets.
const defaultExpirationSeconds = 86400 // = 1 day

// TargetAccess bundles the various objects to access a target cluster.
type TargetAccess struct {
	targetClient     client.Client
	targetRestConfig *rest.Config
	targetClientSet  kubernetes.Interface
}

func (ta *TargetAccess) TargetClient() client.Client {
	return ta.targetClient
}

func (ta *TargetAccess) TargetRestConfig() *rest.Config {
	return ta.targetRestConfig
}

func (ta *TargetAccess) TargetClientSet() kubernetes.Interface {
	return ta.targetClientSet
}

// NewTargetAccess constructs a TargetAccess, handling the different subtypes of kubernetes-cluster Targets, namely:
// - Targets with a kubeconfig,
// - OIDC Targets, and
// - Self Targets, i.e. Targets pointing to the resource cluster watched by the Landscaper.
func NewTargetAccess(ctx context.Context, resolvedTarget *lsv1alpha1.ResolvedTarget,
	lsUncachedClient client.Client, lsRestConfig *rest.Config) (_ *TargetAccess, err error) {

	if resolvedTarget == nil {
		return nil, errors.New("no target defined")
	}

	if resolvedTarget.Target == nil {
		return nil, fmt.Errorf("resolved target does not contain the original target")
	}

	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	if err := yaml.Unmarshal([]byte(resolvedTarget.Content), targetConfig); err != nil {
		return nil, fmt.Errorf("unable to parse target confíguration: %w", err)
	}

	var restConfig *rest.Config
	if targetConfig.Kubeconfig.StrVal != nil {
		kubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)
		restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return nil, fmt.Errorf("unable to create rest config from kubeconfig: %w", err)
		}

	} else if targetConfig.OIDCConfig != nil {
		restConfig, err = getRestConfigForOIDCTarget(ctx, targetConfig.OIDCConfig, resolvedTarget, lsUncachedClient)
		if err != nil {
			return nil, err
		}

	} else if targetConfig.SelfConfig != nil {
		restConfig, err = getRestConfigForSelfTarget(ctx, targetConfig.SelfConfig, resolvedTarget, lsUncachedClient, lsRestConfig)
		if err != nil {
			return nil, err
		}

	} else {
		return nil, fmt.Errorf("target contains neither kubeconfig, nor oidc config, nor self config")
	}

	targetClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	targetClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &TargetAccess{
		targetClient:     targetClient,
		targetRestConfig: restConfig,
		targetClientSet:  targetClientSet,
	}, nil
}

func getRestConfigForOIDCTarget(ctx context.Context, oidcConfig *targettypes.OIDCConfig, resolvedTarget *lsv1alpha1.ResolvedTarget, lsUncachedClient client.Client) (*rest.Config, error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: resolvedTarget.Namespace,
			Name:      oidcConfig.ServiceAccount.Name,
		},
	}

	expirationSeconds := oidcConfig.ExpirationSeconds
	if expirationSeconds == nil {
		expirationSeconds = ptr.To[int64](defaultExpirationSeconds)
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         oidcConfig.Audience,
			ExpirationSeconds: expirationSeconds,
		},
	}

	if err := lsUncachedClient.SubResource("token").Create(ctx, serviceAccount, tokenRequest); err != nil {
		return nil, fmt.Errorf("unable to create token for oidc target: %w", err)
	}

	return &rest.Config{
		Host:        oidcConfig.Server,
		BearerToken: tokenRequest.Status.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: oidcConfig.CAData,
		},
	}, nil
}

func getRestConfigForSelfTarget(ctx context.Context, selfConfig *targettypes.SelfConfig,
	resolvedTarget *lsv1alpha1.ResolvedTarget, lsUncachedClient client.Client, lsRestConfig *rest.Config) (*rest.Config, error) {

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: resolvedTarget.Namespace,
			Name:      selfConfig.ServiceAccount.Name,
		},
	}

	expirationSeconds := selfConfig.ExpirationSeconds
	if expirationSeconds == nil {
		expirationSeconds = ptr.To[int64](defaultExpirationSeconds)
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: expirationSeconds,
		},
	}

	if err := lsUncachedClient.SubResource("token").Create(ctx, serviceAccount, tokenRequest); err != nil {
		return nil, fmt.Errorf("unable to create token for self target: %w", err)
	}

	return &rest.Config{
		Host:            lsRestConfig.Host,
		BearerToken:     tokenRequest.Status.Token,
		TLSClientConfig: lsRestConfig.TLSClientConfig,
	}, nil
}
