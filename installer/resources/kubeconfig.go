package resources

import (
	"context"
	"fmt"
	auth "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

type KubeConfig struct {
	APIVersion     string         `json:"apiVersion"`
	Kind           string         `json:"kind"`
	Clusters       []NamedCluster `json:"clusters"`
	Contexts       []NamedContext `json:"contexts"`
	CurrentContext string         `json:"current-context"`
	Users          []NamedUser    `json:"users"`
}

type NamedCluster struct {
	Name    string            `json:"name"`
	Cluster KubeConfigCluster `json:"cluster"`
}

type KubeConfigCluster struct {
	Server                   string `json:"server"`
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
}

type NamedContext struct {
	Name    string            `json:"name"`
	Context KubeConfigContext `json:"context"`
}

type KubeConfigContext struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
}

type NamedUser struct {
	Name string `json:"name"`
	User User   `json:"user"`
}

type User struct {
	Token string `json:"token"`
}

func CreateKubeconfig(ctx context.Context, cluster *Cluster, serviceAccountName, serviceAccountNamespace string) ([]byte, error) {

	token, err := requestToken(ctx, cluster, serviceAccountName, serviceAccountNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to request token for service account %s/%s: %w", serviceAccountNamespace, serviceAccountName, err)
	}

	contextName := fmt.Sprintf("%s-%s", serviceAccountNamespace, serviceAccountName)

	kubeConfig := KubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: contextName,
		Contexts: []NamedContext{
			{
				Name: contextName,
				Context: KubeConfigContext{
					Cluster: contextName,
					User:    contextName,
				},
			},
		},
		Clusters: []NamedCluster{
			{
				Name: contextName,
				Cluster: KubeConfigCluster{
					Server:                   cluster.RestConfig().Host,
					CertificateAuthorityData: cluster.RestConfig().CAData,
				},
			},
		},
		Users: []NamedUser{
			{
				Name: contextName,
				User: User{
					Token: token,
				},
			},
		},
	}

	kubeconfigYaml, err := yaml.Marshal(&kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	return kubeconfigYaml, nil
}

func requestToken(ctx context.Context, cluster *Cluster, serviceAccountName, serviceAccountNamespace string) (string, error) {

	tokenRequest := &auth.TokenRequest{
		Spec: auth.TokenRequestSpec{
			ExpirationSeconds: ptr.To[int64](7776000),
		},
	}

	tokenRequest, err := cluster.ClientSet().CoreV1().ServiceAccounts(serviceAccountNamespace).CreateToken(ctx, serviceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return tokenRequest.Status.Token, nil
}
