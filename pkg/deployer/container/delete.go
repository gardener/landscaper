// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
)

// Delete handles the delete flow for container deploy item.
func (c *Container) Delete(ctx context.Context) error {
	if c.ProviderStatus.LastOperation != string(container.OperationDelete) || !lsv1alpha1helper.IsCompletedExecutionPhase(c.DeployItem.Status.Phase) {
		// do default reconcile until the pod has finished
		return c.Reconcile(ctx, container.OperationDelete)
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
	sa.Namespace = c.DeployItem.Namespace

	role := &rbacv1.Role{}
	role.Name = sa.Name
	role.Namespace = sa.Namespace

	rolebinding := &rbacv1.RoleBinding{}
	rolebinding.Name = sa.Name
	rolebinding.Namespace = sa.Namespace

	if err := c.kubeClient.Delete(ctx, rolebinding); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.kubeClient.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.kubeClient.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	c.log.V(3).Info("successfully removed init container rbac resources")

	sa = &corev1.ServiceAccount{}
	sa.Name = WaitContainerServiceAccountName(c.DeployItem)
	sa.Namespace = c.DeployItem.Namespace

	role = &rbacv1.Role{}
	role.Name = sa.Name
	role.Namespace = sa.Namespace

	rolebinding = &rbacv1.RoleBinding{}
	rolebinding.Name = sa.Name
	rolebinding.Namespace = sa.Namespace

	if err := c.kubeClient.Delete(ctx, rolebinding); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.kubeClient.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := c.kubeClient.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	c.log.V(3).Info("successfully removed wait container rbac resources")

	return nil
}

func (c *Container) cleanupDeployItem(ctx context.Context) error {
	// delete the referenced export secret if there is one
	if c.DeployItem.Status.ExportReference != nil {
		secret := &corev1.Secret{}
		secret.Name = c.DeployItem.Status.ExportReference.Name
		secret.Namespace = c.DeployItem.Status.ExportReference.Namespace
		if err := c.kubeClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	controllerutil.RemoveFinalizer(c.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return c.kubeClient.Update(ctx, c.DeployItem)
}
