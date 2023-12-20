// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// cleanupOrphanedDeployItemsForNewReconcile deletes all orphaned deploy items that are not defined by their execution anymore.
func (o *Operation) cleanupOrphanedDeployItemsForNewReconcile(ctx context.Context, orphaned []*lsv1alpha1.DeployItem) error {
	if len(orphaned) == 0 {
		return nil
	}
	for i := range orphaned {
		item := orphaned[i]
		itemName, ok := item.Labels[lsv1alpha1.ExecutionManagedNameLabel]
		if ok {
			o.exec.Status.ExecutionGenerations = removeExecutionGeneration(o.exec.Status.ExecutionGenerations, itemName)
			if err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000146, o.exec); err != nil {
				msg := fmt.Sprintf("unable to patch execution status %s", o.exec.Name)
				return lserrors.NewWrappedError(err, "cleanupOrphanedDeployItemsForNewReconcile", msg, err.Error())
			}
		}

		if item.DeletionTimestamp.IsZero() {
			if err := o.Writer().DeleteDeployItem(ctx, read_write_layer.W000064, item); err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("unable to delete deploy item %s", item.Name)
				}
			}
		}
	}
	return nil
}
