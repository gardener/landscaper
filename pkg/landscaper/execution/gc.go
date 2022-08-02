// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// cleanupOrphanedDeployItems deletes all orphaned deploy items that are not defined by their execution anymore.
func (o *Operation) cleanupOrphanedDeployItems(ctx context.Context, orphaned []lsv1alpha1.DeployItem) error {
	if len(orphaned) == 0 {
		return nil
	}
	for _, item := range orphaned {
		if item.DeletionTimestamp.IsZero() && o.checkGCDeletable(item, orphaned) {
			if err := o.Writer().DeleteDeployItem(ctx, read_write_layer.W000064, &item); err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("unable to delete deploy item %s", item.Name)
				}
			}
		}
	}
	return fmt.Errorf("waiting for %d orphaned deploy items to be deleted", len(orphaned))
}

// cleanupOrphanedDeployItemsForNewReconcile deletes all orphaned deploy items that are not defined by their execution anymore.
func (o *Operation) cleanupOrphanedDeployItemsForNewReconcile(ctx context.Context, orphaned []lsv1alpha1.DeployItem) error {
	if len(orphaned) == 0 {
		return nil
	}
	for _, item := range orphaned {
		if item.DeletionTimestamp.IsZero() {
			if err := o.Writer().DeleteDeployItem(ctx, read_write_layer.W000064, &item); err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("unable to delete deploy item %s", item.Name)
				}
			}
		}

		itemName, ok := item.Labels[lsv1alpha1.ExecutionManagedNameLabel]
		if ok {
			o.exec.Status.DeployItemReferences = helper.RemoveVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, itemName)
			o.exec.Status.ExecutionGenerations = removeExecutionGeneration(o.exec.Status.ExecutionGenerations, itemName)
			if err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000146, o.exec); err != nil {
				msg := fmt.Sprintf("unable to patch execution status %s", o.exec.Name)
				return lserrors.NewWrappedError(err, "cleanupOrphanedDeployItemsForNewReconcile", msg, err.Error())
			}
		}
	}
	return nil
}

// checkGCDeletable checks whether all deploy items depending on a given deploy item have been successfully deleted.
// only other deleted items are respected.
func (o *Operation) checkGCDeletable(item lsv1alpha1.DeployItem, items []lsv1alpha1.DeployItem) bool {
	for _, di := range items {
		if di.Name == item.Name {
			continue
		}

		dependsOn := sets.NewString(strings.Split(di.Annotations[lsv1alpha1.ExecutionDependsOnAnnotation], ",")...)
		if dependsOn.Has(item.Labels[lsv1alpha1.ExecutionManagedNameLabel]) {
			return false
		}
	}
	return true
}
