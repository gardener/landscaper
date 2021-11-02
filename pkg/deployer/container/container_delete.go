// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	lserrors "github.com/gardener/landscaper/apis/errors"

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

	if err := CleanupRBAC(ctx, c.DeployItem, c.hostClient, c.Configuration.Namespace); err != nil {
		return lserrors.NewWrappedError(err,
			"Delete", "CleanupRBAC", err.Error())
	}
	if err := CleanupDeployItem(ctx, c.DeployItem, c.lsClient, c.directHostClient, c.Configuration.Namespace); err != nil {
		return lserrors.NewWrappedError(err,
			"Delete", "CleanupDeployItem", err.Error())
	}
	return nil
}
