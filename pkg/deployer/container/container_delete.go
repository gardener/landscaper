// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
)

// Delete handles the delete flow for container deploy item.
func (c *Container) Delete(ctx context.Context) error {
	if _, err := timeout.TimeoutExceeded(ctx, c.DeployItem, TimeoutCheckpointContainerStartDelete); err != nil {
		return err
	}

	// skip the deletion container when the force cleanup annotation is set
	if _, ok := c.DeployItem.Annotations[container.ContainerDeployerOperationForceCleanupAnnotation]; !ok {
		if c.ProviderStatus.LastOperation != string(container.OperationDelete) || c.DeployItem.Status.Phase != lsv1alpha1.DeployItemPhases.Succeeded {
			// do default reconcile until the pod has finished
			return c.Reconcile(ctx, container.OperationDelete)
		}
	}

	if err := CleanupRBAC(ctx, c.DeployItem, c.hostUncachedClient, c.Configuration.Namespace); err != nil {
		return lserrors.NewWrappedError(err,
			"Delete", "CleanupRBAC", err.Error())
	}
	if err := CleanupDeployItem(ctx, c.DeployItem, c.lsUncachedClient, c.hostUncachedClient, c.Configuration.Namespace); err != nil {
		return lserrors.NewWrappedError(err,
			"Delete", "CleanupDeployItem", err.Error())
	}
	return nil
}
