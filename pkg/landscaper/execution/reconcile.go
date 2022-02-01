// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func (o *Operation) Reconcile(ctx context.Context) error {
	op := "Reconcile"
	cond := lsv1alpha1helper.GetOrInitCondition(o.exec.Status.Conditions, lsv1alpha1.ReconcileDeployItemsCondition)
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
	logger := o.Log().WithValues("operation", "reconcile", "resource", kutil.ObjectKeyFromObject(o.exec).String())

	managedItems, err := o.listManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListManagedDeployItems", err.Error())
	}
	executionItems, orphaned := o.getExecutionItems(managedItems)
	if err := o.cleanupOrphanedDeployItems(ctx, orphaned); err != nil {
		return lserrors.NewWrappedError(err, op, "CleanupOrphanedDeployItems", err.Error())
	}

	var phase lsv1alpha1.ExecutionPhase
	for _, item := range executionItems {
		if item.DeployItem != nil {
			dlogger := logger.WithValues("deployitem", item.Info.Name)
			gen := o.getGenerations(item)

			deployItemUpToDate := gen.IsUpToDate()

			if gen.HasExecutionBeenModified() {
				dlogger.V(7).Info("execution has been changed since deployitem has last been applied", "executionGenerationInExecution", gen.ExecutionGenerationInExecution, "executionGenerationInDeployItem", gen.ExecutionGenerationInDeployItem)
			}
			if gen.HasDeployItemBeenModified() {
				dlogger.V(7).Info("deployitem has been modified since the execution has last seen it", "deployItemGeneration", gen.DeployItemGenerationInDeployItem, "lastSeenGeneration", gen.DeployItemGenerationInExecution)
			}

			if deployItemUpToDate && !o.forceReconcile {
				phase = lsv1alpha1helper.CombinedExecutionPhase(phase, item.DeployItem.Status.Phase)
				deployItemStatusUpToDate := item.DeployItem.Status.ObservedGeneration == item.DeployItem.GetGeneration()

				// we need to check if the deployitem is failed due to that its not picked up by a deployer.
				// This is a error scenario that needs special handling as other detections in this code will not work.
				// Mostly due to outdated observed generation
				if item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed &&
					(item.DeployItem.Status.LastError != nil && item.DeployItem.Status.LastError.Reason == lsv1alpha1.PickupTimeoutReason) {
					o.exec.Status.ObservedGeneration = o.exec.Generation
					o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
					o.exec.Status.Conditions = lsv1alpha1helper.MergeConditions(o.exec.Status.Conditions,
						lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
							"DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) has not been picked up by a Deployer", item.Info.Name, item.DeployItem.Name)))
					o.exec.Status.LastError = lserrors.UpdatedError(
						o.exec.Status.LastError,
						"DeployItemReconcile",
						"DeployItemFailed",
						fmt.Sprintf("DeployItem %s (%s) has not been picked up by a Deployer", item.Info.Name, item.DeployItem.Name),
					)
					dlogger.V(7).Info("deployitem in pickup timeout, aborting reconcile")
					return nil
				}

				// deployitem is running - either in a non-final phase, or its observedGeneration doesn't match its generation
				if !lsv1alpha1helper.IsCompletedExecutionPhase(item.DeployItem.Status.Phase) || !deployItemStatusUpToDate {
					o.EventRecorder().Eventf(o.exec, corev1.EventTypeNormal,
						"DeployItemCompleted",
						"deployitem %s not triggered because it already exists and is not completed", item.Info.Name,
					)
					dlogger.V(7).Info("deployitem not triggered because already existing and not completed")
					o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
					phase = lsv1alpha1.ExecutionPhaseProgressing
					continue
				}

				// the deployitem up-to-date
				if deployItemUpToDate {

					// deployitem is failed => set execution to failed
					if item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
						// TODO: check if need to wait for other deployitems to finish
						o.exec.Status.ObservedGeneration = o.exec.Generation
						o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
						o.exec.Status.Conditions = lsv1alpha1helper.MergeConditions(o.exec.Status.Conditions,
							lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
								"DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) is in failed state", item.Info.Name, item.DeployItem.Name)))
						o.exec.Status.LastError = lserrors.UpdatedError(
							o.exec.Status.LastError,
							"DeployItemReconcile",
							"DeployItemFailed",
							fmt.Sprintf("reconciliation of deployitem %q failed", item.DeployItem.Name),
						)
						dlogger.V(7).Info("deployitem failed, aborting reconcile")
						return nil
					}

					// the deployitem is
					// - up-to-date
					// - in a final state
					// - not failed
					// => nothing to do with the deployitem
					dlogger.V(7).Info("deployitem not triggered because up-to-date", "deployItemPhase", string(item.DeployItem.Status.Phase))
					continue
				}
			}
		}
		runnable, err := o.checkRunnable(ctx, item, executionItems)
		if err != nil {
			return lserrors.NewWrappedError(err,
				"CheckReconcilable",
				fmt.Sprintf("check if deployitem %q is able to be reconciled", item.DeployItem.Name),
				err.Error(),
			)
		}
		if !runnable {
			o.Log().V(5).Info("deployitem not runnable", "name", item.Info.Name)
			continue
		}
		if err := o.deployOrTrigger(ctx, item); err != nil {
			msg := fmt.Sprintf("error while creating deployitem %q", item.Info.Name)
			if item.DeployItem != nil {
				msg = fmt.Sprintf("error while triggering deployitem %s", item.DeployItem.Name)
			}
			return lserrors.NewWrappedError(err, "TriggerDeployItem", msg, err.Error())
		}
		phase = lsv1alpha1helper.CombinedExecutionPhase(phase, lsv1alpha1.ExecutionPhaseInit)
	}

	// remove force annotation
	if o.forceReconcile {
		old := o.exec.DeepCopy()
		delete(o.exec.Annotations, lsv1alpha1.OperationAnnotation)
		if err := o.Client().Patch(ctx, o.exec, client.MergeFrom(old)); err != nil {
			o.EventRecorder().Event(o.exec, corev1.EventTypeWarning, "RemoveForceReconcileAnnotation", err.Error())
			return lserrors.NewWrappedError(err, op, "RemoveForceReconcileAnnotation", err.Error())
		}
	}

	if !lsv1alpha1helper.IsCompletedExecutionPhase(phase) {
		return nil
	}

	if err := o.collectAndUpdateExports(ctx, executionItems); err != nil {
		return lserrors.NewWrappedError(err, op, "CollectAndUpdateExports", err.Error())
	}

	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	o.exec.Status.Conditions = lsv1alpha1helper.MergeConditions(o.exec.Status.Conditions,
		lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
			"DeployItemsReconciled", "All DeployItems are successfully reconciled"))
	return nil
}

// deployOrTrigger creates a new deployitem or triggers it if it already exists.
func (o *Operation) deployOrTrigger(ctx context.Context, item executionItem) error {
	if item.DeployItem == nil {
		item.DeployItem = &lsv1alpha1.DeployItem{}
		item.DeployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Info.Name)
		item.DeployItem.Namespace = o.exec.Namespace
	}
	item.DeployItem.Spec.RegistryPullSecrets = o.exec.Spec.RegistryPullSecrets

	if _, err := kutil.CreateOrUpdate(ctx, o.Client(), item.DeployItem, func() error {
		ApplyDeployItemTemplate(item.DeployItem, item.Info)
		kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedByLabel, o.exec.Name)
		item.DeployItem.Spec.Context = o.exec.Spec.Context
		o.Scheme().Default(item.DeployItem)
		return controllerutil.SetControllerReference(o.exec, item.DeployItem, o.Scheme())
	}); err != nil {
		return err
	}

	ref := lsv1alpha1.VersionedNamedObjectReference{}
	ref.Name = item.Info.Name
	ref.Reference.Name = item.DeployItem.Name
	ref.Reference.Namespace = item.DeployItem.Namespace
	ref.Reference.ObservedGeneration = item.DeployItem.Generation

	old := o.exec.DeepCopy()
	o.exec.Status.DeployItemReferences = lsv1alpha1helper.SetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, ref)
	o.exec.Status.ExecutionGenerations = setExecutionGeneration(o.exec.Status.ExecutionGenerations, item.Info.Name, o.exec.Generation)
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
	if err := o.Client().Status().Patch(ctx, o.exec, client.MergeFrom(old)); err != nil {
		return fmt.Errorf("unable to patch deployitem status: %w", err)
	}
	return nil
}

// collectAndUpdateExports loads all exports of all deployitems and
// persists them in a data object in the cluster.
// It also updates the export reference of the execution.
func (o *Operation) collectAndUpdateExports(ctx context.Context, items []executionItem) error {
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
func (o *Operation) checkRunnable(ctx context.Context, item executionItem, items []executionItem) (bool, error) {
	if len(item.Info.DependsOn) == 0 {
		return true, nil
	}

	for _, dep := range item.Info.DependsOn {
		found := false
		for _, exec := range items {
			if exec.Info.Name != dep {
				continue
			}
			found = true
			if exec.DeployItem == nil { // dependent deployitem has never run
				return false, nil
			}
			gen := o.getGenerations(exec)
			if !gen.IsUpToDate() { // dependent deployitem not up-to-date
				return false, nil
			}
			if exec.DeployItem.Status.ObservedGeneration != exec.DeployItem.Generation { // dependent deployitem status not up-to-date
				return false, nil
			}
			if exec.DeployItem.Status.Phase != lsv1alpha1.ExecutionPhaseSucceeded { // dependent deployitem not finished
				return false, nil
			}
			break
		}
		if !found {
			return false, fmt.Errorf("dependent deployitem '%s' not found", dep)
		}
	}
	return true, nil
}

// generations is a helper struct to store generation and observedGeneration values for an execution and one of its deployitems
type generations struct {
	// ExecutionGenerationInExecution is metadata.generation of the execution.
	ExecutionGenerationInExecution int64
	// ExecutionGenerationInDeployItem is the generation which the execution had when it last applied the deployitem. It is stored in the execution status.
	ExecutionGenerationInDeployItem int64
	// is the generation which the deployitem had when the execution last updated it. It is stored in the execution status.
	DeployItemGenerationInExecution int64
	// DeployItemGenerationInDeployItem is metadata.generation of the deployitem.
	DeployItemGenerationInDeployItem int64
}

// IsUpToDate returns whether the deployitem is up-to-date.
// It will return true if both hasExecutionBeenModified() and hasDeployItemBeenModified() return false, and false otherwise.
func (g generations) IsUpToDate() bool {
	return !(g.HasExecutionBeenModified() || g.HasDeployItemBeenModified())
}

// HasExecutionBeenModified returns true if the execution has been modified since the deployitem has last been updated, and false otherwise.
func (g generations) HasExecutionBeenModified() bool {
	return g.ExecutionGenerationInExecution != g.ExecutionGenerationInDeployItem
}

// HasDeployItemBeenModified returns true if the deployitem has been modified since the execution last updated it, and false otherwise.
func (g generations) HasDeployItemBeenModified() bool {
	return g.DeployItemGenerationInDeployItem != g.DeployItemGenerationInExecution
}

// getGenerations returns a generations struct containing the generations and observedGenerations for the execution and the given deployitem
func (o *Operation) getGenerations(item executionItem) generations {
	var lastSeenGeneration int64
	if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, item.Info.Name); ok {
		lastSeenGeneration = ref.Reference.ObservedGeneration
	}
	var lastAppliedGeneration int64
	if expGen, ok := getExecutionGeneration(o.exec.Status.ExecutionGenerations, item.Info.Name); ok {
		lastAppliedGeneration = expGen.ObservedGeneration
	}
	return generations{
		ExecutionGenerationInExecution:   o.exec.GetGeneration(),
		ExecutionGenerationInDeployItem:  lastAppliedGeneration,
		DeployItemGenerationInExecution:  lastSeenGeneration,
		DeployItemGenerationInDeployItem: item.DeployItem.GetGeneration(),
	}
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
