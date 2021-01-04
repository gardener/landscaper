// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraformer

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// EnsureConfig ensures that the configuration, tfvars and state are created.
func (t *Terraformer) EnsureConfig(ctx context.Context, main, variables, tfvars string) error {
	if err := t.createOrUpdateConfig(ctx, main, variables, tfvars); err != nil {
		return err
	}
	return t.waitForConfig(ctx)
}

// createOrUpdateConfig creates or updates the configuration ConfigMap
// and the variables Secret with the given main configuration, variables and tfvars.
func (t *Terraformer) createOrUpdateConfig(ctx context.Context, main, variables, tfvars string) error {
	if _, err := t.createOrUpdateConfigurationConfigMap(ctx, main, variables); err != nil {
		return err
	}
	if _, err := t.createOrUpdateTFVarsSecret(ctx, tfvars); err != nil {
		return err
	}
	if _, err := t.createStateConfigMap(ctx); err != nil {
		return err
	}
	return nil
}

// createOrUpdateTFVarsSecret creates or updates the variables Secret with the given tfvars.
func (t *Terraformer) createOrUpdateTFVarsSecret(ctx context.Context, tfvars string) (*corev1.Secret, error) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.TFVarsSecretName, Labels: t.Labels}}
	_, err := kutils.CreateOrUpdate(ctx, t.kubeClient, secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[TerraformTFVarsKey] = []byte(tfvars)
		return nil
	})
	return secret, err
}

// createStateConfigMap creates an empty state ConfigMap.
func (t *Terraformer) createStateConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.StateConfigMapName, Labels: t.Labels}}
	_, err := kutils.CreateOrUpdate(ctx, t.kubeClient, configMap, func() error {
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		if _, ok := configMap.Data[TerraformStateKey]; !ok {
			configMap.Data[TerraformStateKey] = ""
		}
		return nil
	})
	return configMap, err
}

// createOrUpdateConfigurationConfigMap creates or updates the configuration ConfigMap
// with the given main conficuration and variables.
func (t *Terraformer) createOrUpdateConfigurationConfigMap(ctx context.Context, main, variables string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.ConfigurationConfigMapName, Labels: t.Labels}}
	values := map[string]string{
		TerraformConfigMainKey: main,
		TerraformConfigVarsKey: variables,
	}
	_, err := kutils.CreateOrUpdate(ctx, t.kubeClient, configMap, func() error {
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		for key, value := range values {
			configMap.Data[key] = value
		}
		return nil
	})
	return configMap, err
}

// waitForConfig waits for the Terraform config resources to be created in the cluster.
func (t *Terraformer) waitForConfig(ctx context.Context) error {
	pollCtx, cancel := context.WithTimeout(ctx, DeadlineCleaning)
	defer cancel()

	return wait.PollImmediateUntil(10*time.Second, func() (done bool, err error) {
		t.log.Info("Waiting for Configuration to be created...")
		variablesKey := kutils.ObjectKey(t.TFVarsSecretName, t.Namespace)
		if err = t.kubeClient.Get(pollCtx, variablesKey, &corev1.Secret{}); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get variables secret", "secret", variablesKey.String())
			return false, err
		}

		configurationKey := kutils.ObjectKey(t.ConfigurationConfigMapName, t.Namespace)
		if err = t.kubeClient.Get(pollCtx, configurationKey, &corev1.ConfigMap{}); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get configuration configmap", "configmap", configurationKey.String())
			return false, err
		}

		stateKey := kutils.ObjectKey(t.StateConfigMapName, t.Namespace)
		if err = t.kubeClient.Get(pollCtx, stateKey, &corev1.ConfigMap{}); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get state configmap", "configmap", stateKey.String())
			return false, err
		}
		return true, nil
	}, pollCtx.Done())
}

// cleanUpConfig deletes the two ConfigMaps which store the configuration and state.
// It also deletes the Secret which stores the variables.
func (t *Terraformer) cleanUpConfig(ctx context.Context) error {
	t.log.Info("Cleaning up terraformer configuration")
	t.log.V(1).Info("Deleting Terraform variables Secret", "name", t.TFVarsSecretName)
	err := t.kubeClient.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.TFVarsSecretName, Labels: t.Labels}})
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	t.log.V(1).Info("Deleting Terraform configuration ConfigMap", "name", t.ConfigurationConfigMapName)
	err = t.kubeClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.ConfigurationConfigMapName, Labels: t.Labels}})
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	t.log.V(1).Info("Deleting Terraform state ConfigMap", "name", t.StateConfigMapName)
	err = t.kubeClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.StateConfigMapName, Labels: t.Labels}})
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	return nil
}
