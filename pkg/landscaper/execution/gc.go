// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"
	"strings"

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
			if err := o.Client().Delete(ctx, &item); err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("unable to delete deploy item %s", item.Name)
				}
			}
		}
	}
	return fmt.Errorf("waiting for %d orphaned deploy items to be deleted", len(orphaned))
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
