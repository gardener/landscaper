// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Delete handles the delete flow for a execution
func (o *Operation) DeleteExec(ctx context.Context) error {
	op := "Deletion"
	// set state to deleting
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	managedItems, err := o.listManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}

	if err := o.propagateDeleteWithoutUninstallAnnotation(ctx, managedItems); err != nil {
		return err
	}

	// todo: remove orphaned items and also remove them from the status
	executionItems, _ := o.getExecutionItems(managedItems)

	allDeleted := true
	for _, item := range executionItems {
		if item.DeployItem == nil {
			continue
		}

		gone, err := o.deleteItem(ctx, &item, executionItems)
		if err != nil {
			return err
		}

		allDeleted = allDeleted && gone
	}

	if !allDeleted {
		return nil
	}

	controllerutil.RemoveFinalizer(o.exec, lsv1alpha1.LandscaperFinalizer)
	return lserrors.NewErrorOrNil(read_write_layer.UpdateExecution(ctx, o.Client(), o.exec), op, "RemoveFinalizer")
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

func (o *Operation) deleteItem(ctx context.Context, item *executionItem, executionItems []executionItem) (gone bool, err error) {
	if item.DeployItem == nil {
		return true, nil
	}

	if item.DeployItem.DeletionTimestamp.IsZero() && o.checkDeletable(*item, executionItems) {
		if err := read_write_layer.DeleteDeployItem(ctx, o.Client(), item.DeployItem); err != nil {
			if !apierrors.IsNotFound(err) {
				return false, lserrors.NewWrappedError(err,
					"DeleteDeployItem",
					fmt.Sprintf("unable to delete deploy item %s of step %s", item.DeployItem.Name, item.Info.Name),
					err.Error(),
				)
			}

			return true, nil
		}

		return false, nil
	}

	if !item.DeployItem.DeletionTimestamp.IsZero() && item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
		o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	}

	return false, nil
}

func (o *Operation) propagateDeleteWithoutUninstallAnnotation(ctx context.Context, deployItems []lsv1alpha1.DeployItem) error {
	op := "PropagateDeleteWithoutUninstallAnnotationToDeployItems"

	if !lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(o.exec.ObjectMeta) {
		return nil
	}

	for _, di := range deployItems {
		metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
		if err := read_write_layer.UpdateDeployItem(ctx, o.Client(), &di); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			msg := fmt.Sprintf("unable to add delete-without-uninstall annotation to deploy item %s: %s", di.Name, err.Error())
			return lserrors.NewWrappedError(err, op, "Update", msg)
		}
	}

	return nil
}
