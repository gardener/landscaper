// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Delete handles the delete flow for a execution
func (o *Operation) Delete(ctx context.Context) error {
	op := "Deletion"
	// set state to deleting
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	managedItems, err := o.listManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}
	// todo: remove orphaned items and also remove them from the status
	executionItems, _ := o.getExecutionItems(managedItems)

	allDeleted := true
	for _, item := range executionItems {
		if item.DeployItem == nil {
			continue
		}
		allDeleted = false

		if item.DeployItem.DeletionTimestamp.IsZero() && o.checkDeletable(item, executionItems) {
			if err := o.Client().Delete(ctx, item.DeployItem); err != nil {
				if !apierrors.IsNotFound(err) {
					return lserrors.NewWrappedError(err,
						"DeleteDeployItem",
						fmt.Sprintf("unable to delete deploy item %s of step %s", item.DeployItem.Name, item.Info.Name),
						err.Error(),
					)
				}
				allDeleted = true
			}
			continue
		}

		if item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
			o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		}
	}

	if !allDeleted {
		return nil
	}

	controllerutil.RemoveFinalizer(o.exec, lsv1alpha1.LandscaperFinalizer)
	return lserrors.NewErrorOrNil(o.Client().Update(ctx, o.exec), op, "RemoveFinalizer")
}

// checkDeletable checks whether all deploy items depending on a given deploy item have been successfully deleted.
func (o *Operation) checkDeletable(item executionItem, items []executionItem) bool {
	for _, exec := range items {
		if exec.Info.Name == item.Info.Name {
			continue
		}
		dependsOn := sets.NewString(exec.Info.DependsOn...)
		if !dependsOn.Has(item.Info.Name) {
			continue
		}

		// it is expected that the deploy item is already deleted as all deploy items are listed at the beginning of
		// the reconcile loop.
		// Therefore, it should not be necessary to check again against the api server.
		if exec.DeployItem != nil {
			o.Log().V(3).Info("deploy item %s depends on %s and is still present", exec.DeployItem.Name, item.Info.Name)
			return false
		}
	}
	return true
}
