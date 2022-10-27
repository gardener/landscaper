package targetsync

import (
	"context"
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getSourceClient(ctx context.Context, targetSync *lsv1alpha1.TargetSync, targetClient client.Client,
	schema *runtime.Scheme) (client.Client, error) {

	restConfig, err := getSourceRestConfig(ctx, targetSync, targetClient)
	if err != nil {
		return nil, err
	}

	cl, err := client.New(restConfig, client.Options{
		Scheme: schema,
	})
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func getSourceRestConfig(ctx context.Context, targetSync *lsv1alpha1.TargetSync, targetClient client.Client) (*rest.Config, error) {
	kubeconfigBytes, err := resolveSecretRef(ctx, targetClient, targetSync.Spec.SecretRef, targetSync.Namespace)
	if err != nil {
		return nil, err
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func resolveSecretRef(ctx context.Context, targetClient client.Client,
	secretRef lsv1alpha1.LocalSecretReference, namespace string) ([]byte, error) {

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: namespace,
		Name:      secretRef.Name,
	}

	if err := targetClient.Get(ctx, secretKey, secret); err != nil {
		return nil, err
	}

	kubeconfigBytes := secret.Data[secretRef.Key]
	if len(kubeconfigBytes) == 0 {
		return nil, fmt.Errorf("no kubeconfig in secret")
	}

	return kubeconfigBytes, nil
}
