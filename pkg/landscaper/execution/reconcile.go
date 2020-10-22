// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// executionItem is the internal representation of a execution item with its deployitem and status
type executionItem struct {
	Info       lsv1alpha1.DeployItemTemplate
	DeployItem *lsv1alpha1.DeployItem
}

// Reconcile contains the reconcile logic for a execution item that schedules multiple DeployItems.
func (o *Operation) Reconcile(ctx context.Context) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.exec.Status.Conditions, lsv1alpha1.ReconcileDeployItemsCondition)

	managedItems, err := o.listManagedDeployItems(ctx)
	if err != nil {
		return fmt.Errorf("unable to list managed deploy items: %w", err)
	}
	executionItems, orphaned := o.getExecutionItems(managedItems)
	if err := o.cleanupOrphanedDeployItems(ctx, orphaned); err != nil {
		return err
	}

	var phase lsv1alpha1.ExecutionPhase
	for _, item := range executionItems {
		if item.DeployItem != nil && !o.forceReconcile {
			phase = lsv1alpha1helper.CombinedExecutionPhase(phase, item.DeployItem.Status.Phase)
			if !lsv1alpha1helper.IsCompletedExecutionPhase(item.DeployItem.Status.Phase) {
				o.Log().V(5).Info("deploy item not triggered because already existing and not completed", "name", item.Info.Name)
				o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
				continue
			}
			if item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
				cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) is in failed state", item.Info.Name, item.DeployItem.Name))
				// TODO: check if need to wait for other deploy items to finish
				return o.UpdateStatus(ctx, lsv1alpha1.ExecutionPhaseFailed, cond)
			}
			// get last applied status from own status
			var lastAppliedGeneration int64
			if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, item.Info.Name); ok {
				lastAppliedGeneration = ref.Reference.ObservedGeneration
			}
			if lastAppliedGeneration == o.exec.Generation {
				continue
			}
		}
		runnable, err := o.checkRunnable(ctx, item, executionItems)
		if err != nil {
			return fmt.Errorf("unable to check runnable condition for %s: %w", item.Info.Name, err)
		}
		if !runnable {
			o.Log().V(5).Info("deploy item not runnable", "name", item.Info.Name)
			continue
		}
		if err := o.deployOrTrigger(ctx, item); err != nil {
			return fmt.Errorf("error while triggering deployitem %s: %w", item.Info.Name, err)
		}
		phase = lsv1alpha1helper.CombinedExecutionPhase(phase, lsv1alpha1.ExecutionPhaseInit)
	}

	if !lsv1alpha1helper.IsCompletedExecutionPhase(phase) {
		return nil
	}

	if err := o.collectAndUpdateExports(ctx, executionItems); err != nil {
		return err
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"DeployItemsReconciled", "All DeployItems are successfully reconciled")
	o.exec.Status.ObservedGeneration = o.exec.Generation
	return o.UpdateStatus(ctx, phase, cond)
}

// deployOrTrigger creates a new deploy item or triggers it if it already exists.
func (o *Operation) deployOrTrigger(ctx context.Context, item executionItem) error {

	if item.DeployItem == nil {
		item.DeployItem = &lsv1alpha1.DeployItem{}
		item.DeployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Info.Name)
		item.DeployItem.Namespace = o.exec.Namespace
	}

	if _, err := kutil.CreateOrUpdate(ctx, o.Client(), item.DeployItem, func() error {
		lsv1alpha1helper.SetOperation(&item.DeployItem.ObjectMeta, lsv1alpha1.ReconcileOperation)
		item.DeployItem.Spec.Type = item.Info.Type
		item.DeployItem.Spec.Target = item.Info.Target
		item.DeployItem.Spec.Configuration = item.Info.Configuration
		for k, v := range item.Info.Labels {
			kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, k, v)
		}
		kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedByLabel, o.exec.Name)
		kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedNameLabel, item.Info.Name)
		metav1.SetMetaDataAnnotation(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionDependsOnAnnotation, strings.Join(item.Info.DependsOn, ","))
		return controllerutil.SetControllerReference(o.exec, item.DeployItem, o.Scheme())
	}); err != nil {
		return err
	}

	ref := lsv1alpha1.VersionedNamedObjectReference{}
	ref.Name = item.Info.Name
	ref.Reference.Name = item.DeployItem.Name
	ref.Reference.Namespace = item.DeployItem.Namespace
	ref.Reference.ObservedGeneration = o.exec.Generation

	o.exec.Status.DeployItemReferences = lsv1alpha1helper.SetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, ref)
	return o.UpdateStatus(ctx, lsv1alpha1.ExecutionPhaseProgressing)
}

// collectAndUpdateExports loads all exports of all deploy items and
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

// addExports loads the exports of a deploy item and adds it to the given values.
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

// checkRunnable checks whether all deploy items a given deploy item depends on have been successfully executed.
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
			if exec.DeployItem == nil { // dependent deploy item has never run
				return false, nil
			}
			var lastAppliedGeneration int64
			// TODO: check generation increment or reconcile annotation
			if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, exec.Info.Name); ok {
				lastAppliedGeneration = ref.Reference.ObservedGeneration
			}
			if o.exec.Generation != lastAppliedGeneration { // dependent deploy item not up-to-date
				return false, nil
			}
			if exec.DeployItem.Status.Phase != lsv1alpha1.ExecutionPhaseSucceeded { // dependent deploy item not finished
				return false, nil
			}
			break
		}
		if !found {
			return false, fmt.Errorf("dependent deploy item '%s' not found", dep)
		}
	}
	return true, nil
}
