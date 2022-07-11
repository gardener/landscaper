// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// executionItem is the internal representation of a execution item with its deployitem and status
type executionItem struct {
	Info       lsv1alpha1.DeployItemTemplate
	DeployItem *lsv1alpha1.DeployItem
}

// Reconcile contains the reconcile logic for a execution item that schedules multiple DeployItems.
func (o *Operation) Reconcile(ctx context.Context) lserrors.LsError {
	op := "Reconcile"
	cond := lsv1alpha1helper.GetOrInitCondition(o.exec.Status.Conditions, lsv1alpha1.ReconcileDeployItemsCondition)
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	logger := o.Log().WithValues("operation", "reconcile", "resource", kutil.ObjectKeyFromObject(o.exec).String())

	managedItems, err := o.ListManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListManagedDeployItems", err.Error())
	}
	executionItems, orphaned := o.getExecutionItems(managedItems)
	if err := o.cleanupOrphanedDeployItems(ctx, orphaned); err != nil {
		return lserrors.NewWrappedError(err, op, "CleanupOrphanedDeployItems", err.Error())
	}

	allSucceeded := true
	for _, item := range executionItems {
		dlogger := logger.WithValues("deployitem", item.Info.Name)
		if deployItemUpToDateAndNotForceReconcile(o.exec, item.Info.Name, item.DeployItem, o.forceReconcile, dlogger) {
			if failedBecausePickupTimeout(item.DeployItem) {
				// we need to check if the deploy item is failed due to that it is not picked up by a deployer.
				// This is an error scenario that needs special handling as other detections in this code will not work.
				// Mostly due to outdated observed generation
				setPhaseFailedBecausePickupTimeout(o.exec, cond, item.Info.Name, item.DeployItem.Name, dlogger)
				return nil
			} else if failedAndStatusUpToDate(item.DeployItem) {
				// the deployitem up-to-date
				// deployitem is failed => set execution to failed
				setPhaseFailedBecauseFailedDeployItem(o.exec, cond, item.Info.Name, item.DeployItem.Name, dlogger)
				return nil
			} else if notCompletedPhaseOrNotStatusUpToDate(item.DeployItem) {
				// deployitem is running - either in a non-final phase, or its observedGeneration doesn't match its generation
				setPhaseProgressingOfRunningDeployItem(o.exec, item.Info.Name, o.EventRecorder(), dlogger)
				allSucceeded = false
			} else {
				// the deployitem is: up-to-date, in a final state, not failed => deployItem.spec.phase == succeeded => nothing to do with the deployitem
				dlogger.V(7).Info("deployitem not triggered because up-to-date", "deployItemPhase", string(item.DeployItem.Status.Phase))
			}
		} else { // deploy item not up to date or force reconcile
			allSucceeded = false

			runnable, err := o.checkRunnable(ctx, item, executionItems)
			if err != nil {
				return err
			}

			if runnable {
				if err := o.deployOrTrigger(ctx, *item); err != nil {
					return err
				}
			} else {
				o.Log().V(5).Info("deployitem not runnable", "name", item.Info.Name)
			}
		}
	}

	if allSucceeded {
		if err := o.collectAndUpdateExports(ctx, executionItems); err != nil {
			return lserrors.NewWrappedError(err, op, "CollectAndUpdateExports", err.Error())
		}

		o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		o.exec.Status.Conditions = lsv1alpha1helper.MergeConditions(o.exec.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
				"DeployItemsReconciled", "All DeployItems are successfully reconciled"))
	}

	return nil
}

func failedBecausePickupTimeout(deployItem *lsv1alpha1.DeployItem) bool {
	return deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed &&
		(deployItem.Status.LastError != nil && deployItem.Status.LastError.Reason == lsv1alpha1.PickupTimeoutReason)
}

func failedAndStatusUpToDate(deployItem *lsv1alpha1.DeployItem) bool {
	deployItemStatusUpToDate := deployItem.Status.ObservedGeneration == deployItem.GetGeneration()
	return deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed && deployItemStatusUpToDate
}

func notCompletedPhaseOrNotStatusUpToDate(deployItem *lsv1alpha1.DeployItem) bool {
	deployItemStatusUpToDate := deployItem.Status.ObservedGeneration == deployItem.GetGeneration()
	return !lsv1alpha1helper.IsCompletedExecutionPhase(deployItem.Status.Phase) || !deployItemStatusUpToDate
}

func deployItemUpToDateAndNotForceReconcile(exec *lsv1alpha1.Execution, itemInfoName string,
	deployItem *lsv1alpha1.DeployItem, forceReconcile bool, dlogger logr.Logger) bool {
	if deployItem == nil {
		return false
	}

	gen := newGenerations(exec, itemInfoName, deployItem)

	deployItemUpToDate := gen.IsUpToDate()

	if gen.HasExecutionBeenModified() {
		dlogger.V(7).Info("execution has been changed since deployitem has last been applied",
			"executionGenerationInExecution", gen.ExecutionGenerationInExecution, "executionGenerationInDeployItem",
			gen.ExecutionGenerationInDeployItem)
	}
	if gen.HasDeployItemBeenModified() {
		dlogger.V(7).Info("deployitem has been modified since the execution has last seen it", "deployItemGeneration",
			gen.DeployItemGenerationInDeployItem, "lastSeenGeneration", gen.DeployItemGenerationInExecution)
	}

	return deployItemUpToDate && !forceReconcile
}

func setPhaseFailedBecausePickupTimeout(exec *lsv1alpha1.Execution, cond lsv1alpha1.Condition, infoName, deployItemName string,
	dlogger logr.Logger) {
	exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	exec.Status.Conditions = lsv1alpha1helper.MergeConditions(exec.Status.Conditions,
		lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) is in failed state", infoName, deployItemName)))
	exec.Status.LastError = lserrors.UpdatedError(
		exec.Status.LastError,
		"DeployItemReconcile",
		"DeployItemFailed",
		fmt.Sprintf("reconciliation of deployitem %q failed", deployItemName),
	)
	dlogger.V(7).Info("deployitem failed, aborting reconcile")
}

func setPhaseProgressingOfRunningDeployItem(exec *lsv1alpha1.Execution, itemInfoName string, eventRecorder record.EventRecorder,
	dlogger logr.Logger) {
	eventRecorder.Eventf(exec, corev1.EventTypeNormal,
		"DeployItemCompleted",
		"deployitem %s not triggered because it already exists and is not completed", itemInfoName,
	)
	dlogger.V(7).Info("deployitem not triggered because already existing and not completed")
	exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
}

func setPhaseFailedBecauseFailedDeployItem(exec *lsv1alpha1.Execution, cond lsv1alpha1.Condition, infoName, deployItemName string,
	dlogger logr.Logger) {
	exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	exec.Status.Conditions = lsv1alpha1helper.MergeConditions(exec.Status.Conditions,
		lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) is in failed state", infoName, deployItemName)))
	exec.Status.LastError = lserrors.UpdatedError(
		exec.Status.LastError,
		"DeployItemReconcile",
		"DeployItemFailed",
		fmt.Sprintf("reconciliation of deployitem %q failed", deployItemName),
	)
	dlogger.V(7).Info("deployitem failed, aborting reconcile")
}

// deployOrTrigger creates a new deployitem or triggers it if it already exists.
func (o *Operation) deployOrTrigger(ctx context.Context, item executionItem) lserrors.LsError {
	deployItemExists := item.DeployItem != nil

	if !deployItemExists {
		item.DeployItem = &lsv1alpha1.DeployItem{}
		item.DeployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Info.Name)
		item.DeployItem.Namespace = o.exec.Namespace
	}
	item.DeployItem.Spec.RegistryPullSecrets = o.exec.Spec.RegistryPullSecrets

	if _, err := o.Writer().CreateOrUpdateDeployItem(ctx, read_write_layer.W000036, item.DeployItem, func() error {
		ApplyDeployItemTemplate(item.DeployItem, item.Info)
		kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedByLabel, o.exec.Name)
		item.DeployItem.Spec.Context = o.exec.Spec.Context
		o.Scheme().Default(item.DeployItem)
		return controllerutil.SetControllerReference(o.exec, item.DeployItem, o.Scheme())
	}); err != nil {
		msg := fmt.Sprintf("error while creating deployitem %q", item.Info.Name)
		if deployItemExists {
			msg = fmt.Sprintf("error while triggering deployitem %s", item.DeployItem.Name)
		}
		return lserrors.NewWrappedError(err, "TriggerDeployItem", msg, err.Error())
	}

	ref := lsv1alpha1.VersionedNamedObjectReference{}
	ref.Name = item.Info.Name
	ref.Reference.Name = item.DeployItem.Name
	ref.Reference.Namespace = item.DeployItem.Namespace
	ref.Reference.ObservedGeneration = item.DeployItem.Generation

	o.exec.Status.DeployItemReferences = lsv1alpha1helper.SetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, ref)
	o.exec.Status.ExecutionGenerations = setExecutionGeneration(o.exec.Status.ExecutionGenerations, item.Info.Name, o.exec.Generation)
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
	if err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000034, o.exec); err != nil {
		msg := fmt.Sprintf("unable to patch execution status %s", o.exec.Name)
		return lserrors.NewWrappedError(err, "TriggerDeployItem", msg, err.Error())
	}
	return nil
}

// collectAndUpdateExports loads all exports of all deployitems and
// persists them in a data object in the cluster.
// It also updates the export reference of the execution.
func (o *Operation) collectAndUpdateExports(ctx context.Context, items []*executionItem) error {
	values := make(map[string]interface{})
	for _, item := range items {
		data, err := o.addExports(ctx, item.DeployItem)
		if err != nil {
			return err
		}
		values[item.Info.Name] = data
	}

	return o.CreateOrUpdateExportReference(ctx, values)
}

// CollectAndUpdateExportsNew loads all exports of all deployitems and persists them in a data object in the cluster.
// It also updates the export reference of the execution.
func (o *Operation) CollectAndUpdateExportsNew(ctx context.Context) lserrors.LsError {
	op := "CollectAndUpdateExports"

	items, _, lsErr := o.getDeployItems(ctx)
	if lsErr != nil {
		return lsErr
	}

	values := make(map[string]interface{})
	for _, item := range items {
		data, err := o.addExports(ctx, item.DeployItem)
		if err != nil {
			return lserrors.NewWrappedError(err, op, "AddExports", err.Error())
		}
		values[item.Info.Name] = data
	}

	if err := o.CreateOrUpdateExportReference(ctx, values); err != nil {
		return lserrors.NewWrappedError(err, op, "CreateOrUpdateExportReference", err.Error())
	}

	return nil
}

// addExports loads the exports of a deployitem and adds it to the given values.
func (o *Operation) addExports(ctx context.Context, item *lsv1alpha1.DeployItem) (map[string]interface{}, error) {
	if item.Status.ExportReference == nil {
		return nil, nil
	}
	secret := &corev1.Secret{}
	if err := o.Client().Get(ctx, item.Status.ExportReference.NamespacedName(), secret); err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(secret.Data[lsv1alpha1.DataObjectSecretDataKey], &data); err != nil {
		return nil, err
	}
	return data, nil
}

// checkRunnable checks whether all deployitems a given deployitem depends on have been successfully executed.
func (o *Operation) checkRunnable(ctx context.Context, item *executionItem, items []*executionItem) (bool, lserrors.LsError) {
	if len(item.Info.DependsOn) == 0 {
		return true, nil
	}

	for _, dep := range item.Info.DependsOn {
		found := false
		for _, deployItemPair := range items {
			if deployItemPair.Info.Name != dep {
				continue
			}
			found = true
			if deployItemPair.DeployItem == nil { // dependent deployitem has never run
				return false, nil
			}
			gen := newGenerations(o.exec, deployItemPair.Info.Name, deployItemPair.DeployItem)
			if !gen.IsUpToDate() { // dependent deployitem not up-to-date
				return false, nil
			}
			if deployItemPair.DeployItem.Status.ObservedGeneration != deployItemPair.DeployItem.Generation { // dependent deployitem status not up-to-date
				return false, nil
			}
			if deployItemPair.DeployItem.Status.Phase != lsv1alpha1.ExecutionPhaseSucceeded { // dependent deployitem not finished
				return false, nil
			}
			break
		}
		if !found {
			return false, lserrors.NewError("CheckRunnable", "DependentDeployItemNotFound",
				fmt.Sprintf("dependent deployitem %s of deployitem %s not found", dep, item.Info.Name))
		}
	}
	return true, nil
}

func getExecutionGeneration(objects []lsv1alpha1.ExecutionGeneration, name string) (lsv1alpha1.ExecutionGeneration, bool) {
	for _, ref := range objects {
		if ref.Name == name {
			return ref, true
		}
	}
	return lsv1alpha1.ExecutionGeneration{}, false
}

func setExecutionGeneration(objects []lsv1alpha1.ExecutionGeneration, name string, gen int64) []lsv1alpha1.ExecutionGeneration {
	for i, ref := range objects {
		if ref.Name == name {
			objects[i].ObservedGeneration = gen
			return objects
		}
	}
	return append(objects, lsv1alpha1.ExecutionGeneration{Name: name, ObservedGeneration: gen})
}
