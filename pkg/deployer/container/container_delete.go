// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
)

// Delete handles the delete flow for container deploy item.
func (c *Container) Delete(ctx context.Context) error {
	// skip the deletion container when the force cleanup annotation is set
	if _, ok := c.DeployItem.Annotations[container.ContainerDeployerOperationForceCleanupAnnotation]; !ok {
		if c.ProviderStatus.LastOperation != string(container.OperationDelete) || c.DeployItem.Status.Phase != lsv1alpha1.ExecutionPhaseSucceeded {
			// do default reconcile until the pod has finished
			return c.Reconcile(ctx, container.OperationDelete)
		}
	}

	if err := c.cleanupRBAC(ctx); err != nil {
		return err
	}
	return c.cleanupDeployItem(ctx)
}

// cleanupRBAC removes all service accounts, roles and rolebindings that belong to the deploy item
func (c *Container) cleanupRBAC(ctx context.Context) error {
	sa := &corev1.ServiceAccount{}
	sa.Name = InitContainerServiceAccountName(c.DeployItem)
	sa.Namespace = c.Configuration.Namespace

	role := &rbacv1.Role{}
	role.Name = sa.Name
	role.Namespace = sa.Namespace

	rolebinding := &rbacv1.RoleBinding{}
	rolebinding.Name = sa.Name
	rolebinding.Namespace = sa.Namespace

	if err := c.hostClient.Delete(ctx, rolebinding); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.hostClient.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.hostClient.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	c.log.V(3).Info("successfully removed init container rbac resources")

	sa = &corev1.ServiceAccount{}
	sa.Name = WaitContainerServiceAccountName(c.DeployItem)
	sa.Namespace = c.Configuration.Namespace

	role = &rbacv1.Role{}
	role.Name = sa.Name
	role.Namespace = sa.Namespace

	rolebinding = &rbacv1.RoleBinding{}
	rolebinding.Name = sa.Name
	rolebinding.Namespace = sa.Namespace

	if err := c.hostClient.Delete(ctx, rolebinding); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.hostClient.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.hostClient.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	c.log.V(3).Info("successfully removed wait container rbac resources")

	return nil
}

// cleanupDeployItem deletes all secrets from diRec host cluster which belong to diRec deploy item.
func (c *Container) cleanupDeployItem(ctx context.Context) error {
	secrets := []string{
		ConfigurationSecretName(c.DeployItem.Namespace, c.DeployItem.Name),
		ExportSecretName(c.DeployItem.Namespace, c.DeployItem.Name),
		ImagePullSecretName(c.DeployItem.Namespace, c.DeployItem.Name),
		ComponentDescriptorPullSecretName(c.DeployItem.Namespace, c.DeployItem.Name),
		BluePrintPullSecretName(c.DeployItem.Namespace, c.DeployItem.Name),
	}

	for _, secretName := range secrets {
		secret := &corev1.Secret{}
		secret.Name = secretName
		secret.Namespace = c.Configuration.Namespace
		if err := c.hostClient.Delete(ctx, secret); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		}
	}

	secret := &corev1.Secret{}
	secret.Name = DeployItemExportSecretName(c.DeployItem.Name)
	secret.Namespace = c.DeployItem.Namespace
	if err := c.lsClient.Delete(ctx, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	controllerutil.RemoveFinalizer(c.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return c.lsClient.Update(ctx, c.DeployItem)
}
