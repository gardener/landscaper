// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"context"
	"fmt"
	"strconv"

	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

// The HelmSecretManager manages secrets which are created by Helm to manage releases.
type HelmSecretManager struct {
	clientset        *kubernetes.Clientset
	defaultNamespace string
}

// NewHelmSecretManager creates a new helm secret manager.
func NewHelmSecretManager(targetRestConfig *rest.Config, defaultNamespace string, logf func(string, ...interface{})) (*HelmSecretManager, error) {
	manager := &HelmSecretManager{}

	restClientGetter := newRemoteRESTClientGetter(targetRestConfig, defaultNamespace)
	kc := kube.New(restClientGetter)
	kc.Log = logf

	clientset, err := kc.Factory.KubernetesClientSet()
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	manager.clientset = clientset
	manager.defaultNamespace = defaultNamespace
	return manager, nil
}

// DeletePendingReleaseSecrets lists all secrets that are created for the given Helm release which are in a pending state.
// The secret latest (the greatest version) release version is being deleted.
func (m *HelmSecretManager) DeletePendingReleaseSecrets(ctx context.Context, releaseName string) error {
	logger, _ := logging.FromContextOrNew(ctx, nil)

	logger.Debug("try delete pending helm releases",
		lc.KeyResource, types.NamespacedName{Name: releaseName, Namespace: m.defaultNamespace})

	secrets, err := m.clientset.CoreV1().Secrets(m.defaultNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("name=%s, status in (pending-install, pending-upgrade)", releaseName),
	})

	if err != nil {
		return fmt.Errorf("failed to list helm secrets: %w", err)
	}

	latestReleaseVersion := 0
	var latestReleaseVersionSecret *corev1.Secret
	var version string
	var hasVersion bool

	for _, secret := range secrets.Items {
		if secret.Labels == nil {
			continue
		}

		version, hasVersion = secret.Labels["version"]

		if !hasVersion {
			continue
		}

		releaseVersion, err := strconv.Atoi(version)
		if err != nil {
			continue
		}

		if releaseVersion > latestReleaseVersion {
			latestReleaseVersion = releaseVersion
			latestReleaseVersionSecret = &secret
		}
	}

	if latestReleaseVersionSecret != nil {
		logger.Info("deleting pending helm release",
			lc.KeyResource, types.NamespacedName{Name: latestReleaseVersionSecret.Name, Namespace: latestReleaseVersionSecret.Namespace})
		err = m.clientset.CoreV1().Secrets(latestReleaseVersionSecret.Namespace).Delete(ctx, latestReleaseVersionSecret.Name, metav1.DeleteOptions{})

		if err != nil {
			return fmt.Errorf("failed to delete helm secret: %w", err)
		}
	}

	return nil
}
