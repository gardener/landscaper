package kubernetes

import (
	"context"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSecretsForServiceAccount returns the list of secrets of type "kubernetes.io/service-account-token"
// which belong to the given service account. The result list is sorted decreasingly by creation timestamp,
// so that it starts with the newest secret.
func GetSecretsForServiceAccount(ctx context.Context, kubeClient client.Client, serviceAccountKey client.ObjectKey) ([]*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := kubeClient.List(ctx, secretList, client.InNamespace(serviceAccountKey.Namespace)); err != nil {
		return nil, err
	}

	result := []*corev1.Secret{}
	for i := range secretList.Items {
		s := &secretList.Items[i]
		if s.Type == corev1.SecretTypeServiceAccountToken {
			serviceAccountName, ok := s.Annotations[corev1.ServiceAccountNameKey]
			if ok && serviceAccountName == serviceAccountKey.Name {
				result = append(result, s)
			}
		}
	}

	// Sort the result list so that it starts with the newest secret
	sort.Slice(result, func(i, j int) bool {
		return result[j].ObjectMeta.CreationTimestamp.Before(&result[i].ObjectMeta.CreationTimestamp)
	})

	return result, nil
}

func CreateSecretForServiceAccount(ctx context.Context, kubeClient client.Client, serviceAccount *corev1.ServiceAccount) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:  map[string]string{corev1.ServiceAccountNameKey: serviceAccount.Name},
			GenerateName: serviceAccount.Name + "-token-",
			Namespace:    serviceAccount.Namespace,
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	if err := kubeClient.Create(ctx, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func WaitForServiceAccountToken(ctx context.Context, kubeClient client.Client, secretKey client.ObjectKey) error {
	return wait.PollImmediate(10*time.Second, 5*time.Minute, func() (done bool, err error) {
		secret := &corev1.Secret{}
		if err := kubeClient.Get(ctx, secretKey, secret); err != nil {
			return false, nil
		}

		if len(secret.Data) == 0 {
			return false, nil
		}

		token, ok := secret.Data[corev1.ServiceAccountTokenKey]
		return ok && len(token) > 0, nil
	})
}
