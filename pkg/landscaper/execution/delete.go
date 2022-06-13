// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

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
func (o *Operation) Delete(ctx context.Context) lserrors.LsError {
	op := "Deletion"
	// set state to deleting
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	managedItems, err := o.ListManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}

	if err := o.propagateDeleteWithoutUninstallAnnotation(ctx, managedItems); err != nil {
		return err
	}

	// todo: remove orphaned items and also remove them from the status
	executionItems, _ := o.getExecutionItems(managedItems)

	allDeleted := true
	oneDeleteFailed := false
	for _, item := range executionItems {
		if item.DeployItem == nil {
			continue
		}

		gone, deleteFailed, err := o.deleteItem(ctx, item, executionItems)
		if err != nil {
			return err
		}

		allDeleted = allDeleted && gone
		oneDeleteFailed = oneDeleteFailed || deleteFailed
	}

	if oneDeleteFailed {
		o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	}

	if !allDeleted {
		return nil
	}

	controllerutil.RemoveFinalizer(o.exec, lsv1alpha1.LandscaperFinalizer)
	err = o.Writer().UpdateExecution(ctx, read_write_layer.W000026, o.exec)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "RemoveFinalizer", err.Error())
	}
	return nil
}

// checkDeletable checks whether all deploy items depending on a given deploy item have been successfully deleted.
func (o *Operation) checkDeletable(item executionItem, items []*executionItem) bool {
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

func (o *Operation) DeleteItemOld(ctx context.Context, item *executionItem, executionItems []*executionItem) (gone bool,
	deleteFailed bool, err lserrors.LsError) {
	if item.DeployItem == nil {
		return true, false, nil
	}

	if item.DeployItem.DeletionTimestamp.IsZero() && o.checkDeletable(*item, executionItems) {
		if err := o.Writer().DeleteDeployItem(ctx, read_write_layer.W000065, item.DeployItem); err != nil {
			if !apierrors.IsNotFound(err) {
				return false, false, lserrors.NewWrappedError(err,
					"DeleteDeployItem",
					fmt.Sprintf("unable to delete deploy item %s of step %s", item.DeployItem.Name, item.Info.Name),
					err.Error(),
				)
			}

			return true, false, nil
		}

		return false, false, nil
	}

	if !item.DeployItem.DeletionTimestamp.IsZero() &&
		item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed &&
		item.DeployItem.Status.ObservedGeneration == item.DeployItem.Generation {
		return false, true, nil
	}

	return false, false, nil
}

func (o *Operation) deleteItem(ctx context.Context, item *executionItem, executionItems []*executionItem) (gone bool,
	deleteFailed bool, err lserrors.LsError) {
	if item.DeployItem == nil {
		return true, false, nil
	}

	if o.triggerDeletionIsRequired(ctx, item, executionItems) {
		if err := o.triggerDeletion(ctx, item); err != nil {
			return false, false, err
		}

		if err := o.markTriggerDeletionAsDone(ctx, item); err != nil {
			return false, false, err
		}

		return false, false, nil
	}

	if !item.DeployItem.DeletionTimestamp.IsZero() &&
		item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed &&
		item.DeployItem.Status.ObservedGeneration == item.DeployItem.Generation {
		return false, true, nil
	}

	return false, false, nil
}

func (o *Operation) triggerDeletionIsRequired(_ context.Context, item *executionItem, executionItems []*executionItem) bool {
	lastAppliedGeneration, ok := getExecutionGeneration(o.exec.Status.ExecutionGenerations, item.Info.Name)
	if ok && lastAppliedGeneration.ObservedGeneration == o.exec.Generation {
		return false
	}

	return o.checkDeletable(*item, executionItems)
}

func (o *Operation) triggerDeletion(ctx context.Context, item *executionItem) lserrors.LsError {
	if item.DeployItem.DeletionTimestamp.IsZero() {
		if err := o.Writer().DeleteDeployItem(ctx, read_write_layer.W000113, item.DeployItem); client.IgnoreNotFound(err) != nil {
			return lserrors.NewWrappedError(err, "DeleteDeployItem",
				fmt.Sprintf("unable to delete deploy item %s of step %s", item.DeployItem.Name, item.Info.Name),
				err.Error(),
			)
		}
	} else {
		// if the deletionTimestamp is already set, re-trigger via reconcile annotation
		if _, err := o.Writer().CreateOrUpdateDeployItem(ctx, read_write_layer.W000080, item.DeployItem, func() error {
			lsv1alpha1helper.SetOperation(&item.DeployItem.ObjectMeta, lsv1alpha1.ReconcileOperation)
			lsv1alpha1helper.SetTimestampAnnotationNow(&item.DeployItem.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp)
			return nil
		}); client.IgnoreNotFound(err) != nil {
			msg := fmt.Sprintf("error while re-triggering deletion of deployitem %q", item.Info.Name)
			return lserrors.NewWrappedError(err, "TriggerDeployItemDeletion", msg, err.Error())
		}
	}

	return nil
}

func (o *Operation) markTriggerDeletionAsDone(ctx context.Context, item *executionItem) lserrors.LsError {
	old := o.exec.DeepCopy()
	o.exec.Status.ExecutionGenerations = setExecutionGeneration(o.exec.Status.ExecutionGenerations, item.Info.Name, o.exec.Generation)
	if err := o.Writer().PatchExecutionStatus(ctx, read_write_layer.W000081, o.exec, old); err != nil {
		msg := fmt.Sprintf("unable to patch execution status %s", o.exec.Name)
		return lserrors.NewWrappedError(err, "MarkTriggerDeletionAsDone", msg, err.Error())
	}
	return nil
}

func (o *Operation) propagateDeleteWithoutUninstallAnnotation(ctx context.Context, deployItems []lsv1alpha1.DeployItem) lserrors.LsError {
	op := "PropagateDeleteWithoutUninstallAnnotationToDeployItems"

	if !lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(o.exec.ObjectMeta) {
		return nil
	}

	for _, di := range deployItems {
		metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
		if err := o.Writer().UpdateDeployItem(ctx, read_write_layer.W000041, &di); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			msg := fmt.Sprintf("unable to add delete-without-uninstall annotation to deploy item %s: %s", di.Name, err.Error())
			return lserrors.NewWrappedError(err, op, "Update", msg)
		}
	}

	return nil
}
