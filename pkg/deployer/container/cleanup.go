// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/apis/deployer/container"
	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
)

// CleanupPod cleans up a pod that was started with the container deployer.
func CleanupPod(ctx context.Context, hostClient client.Client, pod *corev1.Pod, keepPod bool) error {
	// only remove the finalizer if we get the status of the pod
	controllerutil.RemoveFinalizer(pod, container.ContainerDeployerFinalizer)
	if err := hostClient.Update(ctx, pod); err != nil {
		err = fmt.Errorf("unable to remove finalizer from pod: %w", err)
		return lserrors.NewWrappedError(err,
			"CleanupPod", "RemoveFinalizer", err.Error())
	}
	if keepPod {
		return nil
	}
	if err := hostClient.Delete(ctx, pod); err != nil {
		err = fmt.Errorf("unable to delete pod: %w", err)
		return lserrors.NewWrappedError(err,
			"CleanupPod", "DeletePod", err.Error())
	}
	return nil
}

// CleanupRBAC removes all service accounts, roles and rolebindings that belong to the deploy item
func CleanupRBAC(ctx context.Context, deployItem *lsv1alpha1.DeployItem, hostClient client.Client, hostNamespace string) error {
	log := logging.FromContextOrDiscard(ctx)
	sa := &corev1.ServiceAccount{}
	sa.Name = InitContainerServiceAccountName(deployItem)
	sa.Namespace = hostNamespace

	role := &rbacv1.Role{}
	role.Name = sa.Name
	role.Namespace = sa.Namespace

	rolebinding := &rbacv1.RoleBinding{}
	rolebinding.Name = sa.Name
	rolebinding.Namespace = sa.Namespace

	if err := hostClient.Delete(ctx, rolebinding); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := hostClient.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := hostClient.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	log.Debug("successfully removed init container rbac resources")

	sa = &corev1.ServiceAccount{}
	sa.Name = WaitContainerServiceAccountName(deployItem)
	sa.Namespace = hostNamespace

	role = &rbacv1.Role{}
	role.Name = sa.Name
	role.Namespace = sa.Namespace

	rolebinding = &rbacv1.RoleBinding{}
	rolebinding.Name = sa.Name
	rolebinding.Namespace = sa.Namespace

	if err := hostClient.Delete(ctx, rolebinding); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := hostClient.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := hostClient.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	log.Debug("successfully removed wait container rbac resources")

	return nil
}

// CleanupDeployItem deletes all secrets from a host cluster which belong to a deploy item.
func CleanupDeployItem(ctx context.Context, deployItem *lsv1alpha1.DeployItem, lsClient, hostClient client.Client, hostNamespace string) error {
	log := logging.FromContextOrDiscard(ctx)
	secrets := []string{
		ConfigurationSecretName(deployItem.Namespace, deployItem.Name),
		ExportSecretName(deployItem.Namespace, deployItem.Name),
		ImagePullSecretName(deployItem.Namespace, deployItem.Name),
		ComponentDescriptorPullSecretName(deployItem.Namespace, deployItem.Name),
		BluePrintPullSecretName(deployItem.Namespace, deployItem.Name),
	}

	for _, secretName := range secrets {
		secret := &corev1.Secret{}
		secret.Name = secretName
		secret.Namespace = hostNamespace
		if err := hostClient.Delete(ctx, secret); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		}
	}

	// cleanup state
	if err := state.CleanupState(ctx,
		log,
		hostClient,
		hostNamespace,
		lsv1alpha1helper.ObjectReferenceFromObject(deployItem)); err != nil {
		return err
	}

	secret := &corev1.Secret{}
	secret.Name = DeployItemExportSecretName(deployItem.Name)
	secret.Namespace = deployItem.Namespace
	if err := lsClient.Delete(ctx, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
	writer := read_write_layer.NewWriter(log, lsClient)
	return writer.UpdateDeployItem(ctx, read_write_layer.W000038, deployItem)
}
