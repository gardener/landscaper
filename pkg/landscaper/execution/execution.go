// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/clusters"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Operation contains all execution operations
type Operation struct {
	*operation.Operation
	exec           *lsv1alpha1.Execution
	forceReconcile bool
}

// NewOperation creates a new execution operations
func NewOperation(op *operation.Operation, exec *lsv1alpha1.Execution, forceReconcile bool) *Operation {
	return &Operation{
		Operation:      op,
		exec:           exec,
		forceReconcile: forceReconcile,
	}
}

func (o *Operation) UpdateDeployItems(ctx context.Context) lserrors.LsError {
	op := "UpdateDeployItems"

	executionItems, orphaned, lsErr := o.getDeployItems(ctx)
	if lsErr != nil {
		return lsErr
	}

	if err := o.cleanupOrphanedDeployItemsForNewReconcile(ctx, orphaned); err != nil {
		return lserrors.NewWrappedError(err, op, "CleanupOrphanedDeployItems", err.Error())
	}

	for _, item := range executionItems {
		lsErr := o.updateDeployItem(ctx, *item)
		if lsErr != nil {
			return lsErr
		}
	}

	return nil
}

func (o *Operation) TriggerDeployItems(ctx context.Context) (*DeployItemClassification, lserrors.LsError) {
	items, orphaned, lsErr := o.getDeployItems(ctx)
	if lsErr != nil {
		return nil, lsErr
	}

	// Trigger orphaned deploy items
	classificationOfOrphans, lsErr := newDeployItemClassificationForOrphans(o.exec.Status.JobID, orphaned)
	if lsErr != nil {
		return nil, lsErr
	}

	if !classificationOfOrphans.AllSucceeded() {
		// Start the runnable items, provided there are no failed items
		if !classificationOfOrphans.HasFailedItems() {
			deletableItems := classificationOfOrphans.GetRunnableItems()
			for _, item := range deletableItems {
				skip, err := o.skipUninstall(ctx, item.DeployItem)
				if err != nil {
					return nil, err
				}

				if skip {
					if err := o.removeFinalizerFromDeployItem(ctx, item.DeployItem); err != nil {
						return nil, err
					}
				} else {
					if err := o.triggerDeployItem(ctx, item.DeployItem); err != nil {
						return nil, err
					}
				}
			}
		}

		return classificationOfOrphans, nil
	}

	// Trigger new and updated deploy items
	classification, lsErr := newDeployItemClassification(o.exec.Status.JobID, items)
	if lsErr != nil {
		return nil, lsErr
	}

	// Start the runnable items, provided there are no failed items
	if !classification.HasFailedItems() {
		runnableItems := classification.GetRunnableItems()
		for _, item := range runnableItems {
			if err := o.triggerDeployItem(ctx, item.DeployItem); err != nil {
				return nil, err
			}
		}
	}

	return classification, nil
}

func (o *Operation) TriggerDeployItemsForDelete(ctx context.Context) (*DeployItemClassification, lserrors.LsError) {
	op := "TriggerDeployItemsForDelete"

	items, _, lsErr := o.getDeployItems(ctx)
	if lsErr != nil {
		return nil, lsErr
	}

	classification, lsErr := newDeployItemClassificationForDelete(o.exec.Status.JobID, items)
	if lsErr != nil {
		return nil, lsErr
	}

	// If all deploy items have been successfully deleted, remove the finalizer of the execution
	if classification.AllSucceeded() {
		controllerutil.RemoveFinalizer(o.exec, lsv1alpha1.LandscaperFinalizer)
		err := o.Writer().UpdateExecution(ctx, read_write_layer.W000096, o.exec)
		if err != nil {
			return classification, lserrors.NewWrappedError(err, op, "RemoveFinalizer", err.Error())
		}
		return classification, nil
	}

	// Start the runnable items, provided there are no failed items
	if !classification.HasFailedItems() {
		deletableItems := classification.GetRunnableItems()
		for _, item := range deletableItems {
			skip, err := o.skipUninstall(ctx, item.DeployItem)
			if err != nil {
				return nil, err
			}

			if skip {
				if err := o.removeFinalizerFromDeployItem(ctx, item.DeployItem); err != nil {
					return nil, err
				}
			} else {
				if err := o.triggerDeployItem(ctx, item.DeployItem); err != nil {
					return nil, err
				}
			}
		}
	}

	return classification, nil
}

func (o *Operation) triggerDeployItem(ctx context.Context, di *lsv1alpha1.DeployItem) lserrors.LsError {
	op := "TriggerDeployItem"

	key := kutil.ObjectKeyFromObject(di)
	di = &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, o.Client(), key, di); err != nil {
		return lserrors.NewWrappedError(err, op, "GetDeployItem", err.Error())
	}

	di.Status.SetJobID(o.exec.Status.JobID)
	now := metav1.Now()
	di.Status.JobIDGenerationTime = &now
	if err := o.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000090, di); err != nil {
		return lserrors.NewWrappedError(err, op, "UpdateDeployItemStatus", err.Error())
	}

	return nil
}
func (o *Operation) skipUninstall(ctx context.Context, di *lsv1alpha1.DeployItem) (bool, lserrors.LsError) {
	op := "skipUninstall"

	if di.Spec.OnDelete == nil || !di.Spec.OnDelete.SkipUninstallIfClusterRemoved || di.Spec.Target == nil {
		return false, nil
	}

	shootName, ok := di.GetAnnotations()[clusterNameAnnotation]
	if !ok {
		return false, nil
	}

	targetSyncs := &lsv1alpha1.TargetSyncList{}
	if err := o.Client().List(ctx, targetSyncs, client.InNamespace(di.GetNamespace())); err != nil {
		msg := fmt.Sprintf("unable to retrieve targetsync object for namespace%s", di.GetNamespace())
		return false, lserrors.NewWrappedError(err, op, msg, err.Error())
	}
	if len(targetSyncs.Items) != 1 {
		return false, lserrors.NewError(op, "fetchTargetSync", "targetsync not found or not unique")
	}

	tgs := targetSyncs.Items[0]

	sourceClientProvider := clusters.NewDefaultSourceClientProvider()
	shootClient, err := sourceClientProvider.GetSourceShootClient(ctx, &tgs, o.Client())
	if err != nil {
		return false, lserrors.NewError(op, "GetSourceShootClient", "failed to get shoot client for skipUninstall")
	}

	exists, err := shootClient.ExistsShoot(ctx, tgs.Spec.SourceNamespace, shootName)
	if err != nil {
		return false, lserrors.NewError(op, "ExistsShoot", "unable to check whether shoot exists")
	}

	return !exists, nil
}

func (o *Operation) removeFinalizerFromDeployItem(ctx context.Context, di *lsv1alpha1.DeployItem) lserrors.LsError {
	op := "removeFinalizerFromDeployItem"

	key := kutil.ObjectKeyFromObject(di)
	di = &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, o.Client(), key, di); err != nil {
		return lserrors.NewWrappedError(err, op, "GetDeployItem", err.Error())
	}

	updated := controllerutil.RemoveFinalizer(di, lsv1alpha1.LandscaperFinalizer)
	if updated {
		if err := o.Writer().UpdateDeployItem(ctx, read_write_layer.W000033, di); err != nil {
			return lserrors.NewWrappedError(err, op, "UpdateDeployItem", err.Error())
		}
	}

	return nil
}

func (o *Operation) getDeployItems(ctx context.Context) ([]*executionItem, []lsv1alpha1.DeployItem, lserrors.LsError) {
	op := "getDeployItems"

	managedItems, err := o.ListManagedDeployItems(ctx)
	if err != nil {
		return nil, nil, lserrors.NewWrappedError(err, op, "ListManagedDeployItems", err.Error())
	}

	executionItems, orphaned := o.getExecutionItems(managedItems)
	return executionItems, orphaned, nil
}

// UpdateStatus updates the status of a execution
func (o *Operation) UpdateStatus(ctx context.Context, updatedConditions ...lsv1alpha1.Condition) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	o.exec.Status.Conditions = lsv1alpha1helper.MergeConditions(o.exec.Status.Conditions, updatedConditions...)
	if err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000032, o.exec); err != nil {
		logger.Error(err, "unable to set installation status")
		return err
	}
	return nil
}

// CreateOrUpdateExportReference creates or updates a dataobject from a object reference
func (o *Operation) CreateOrUpdateExportReference(ctx context.Context, values interface{}) error {
	do := dataobjects.New().
		SetNamespace(o.exec.Namespace).
		SetSource(lsv1alpha1helper.DataObjectSourceFromExecution(o.exec)).
		SetContext(lsv1alpha1helper.DataObjectSourceFromExecution(o.exec)).
		SetData(values)

	raw, err := do.Build()
	if err != nil {
		return err
	}

	if _, err := o.Writer().CreateOrUpdateDataObject(ctx, read_write_layer.W000075, raw, func() error {
		if err := controllerutil.SetOwnerReference(o.exec, raw, api.LandscaperScheme); err != nil {
			return err
		}
		return do.Apply(raw)
	}); err != nil {
		return err
	}

	o.exec.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      raw.Name,
		Namespace: raw.Namespace,
	}
	return o.UpdateStatus(ctx)
}
