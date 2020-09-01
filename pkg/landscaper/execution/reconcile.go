// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package execution

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// executionItem is the internal representation of a execution item with its deployitem and status
type executionItem struct {
	Info       lsv1alpha1.ExecutionItem
	DeployItem *lsv1alpha1.DeployItem
}

// Reconcile contains the reconcile logic for a execution item that schedules multiple DeployItems.
func (o *Operation) Reconcile(ctx context.Context) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.exec.Status.Conditions, lsv1alpha1.ReconcileDeployItemsCondition)

	// todo: make it possible to specify a dag
	managedItems, err := o.listManagedDeployItems(ctx)
	if err != nil {
		return fmt.Errorf("unable to list managed deploy items: %w", err)
	}
	// todo: remove orphaned items and also remove them from the status
	executionItems, _ := o.getExecutionItems(managedItems)

	var phase lsv1alpha1.ExecutionPhase
	for _, item := range executionItems {
		if item.DeployItem == nil {
			return o.deployOrTrigger(ctx, item)
		}
		phase = lsv1alpha1helper.CombinedExecutionPhase(phase, item.DeployItem.Status.Phase)

		// get last applied status from own status
		var lastAppliedGeneration int64
		if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, item.Info.Name); ok {
			lastAppliedGeneration = ref.Reference.ObservedGeneration
		}

		if !lsv1alpha1helper.IsCompletedExecutionPhase(item.DeployItem.Status.Phase) {
			if !lsv1alpha1helper.IsProgressingExecutionPhase(phase) && lastAppliedGeneration != o.exec.Generation {
				return o.deployOrTrigger(ctx, item)
			}
			o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
			return nil
		}

		if item.DeployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
			cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) is in failed state", item.Info.Name, item.DeployItem.Name))
			return o.UpdateStatus(ctx, lsv1alpha1.ExecutionPhaseFailed, cond)
		}

		// we already updated this item
		if lastAppliedGeneration == o.exec.Generation {
			continue
		}
		continue
	}

	if err := o.collectAndUpdateExports(ctx); err != nil {
		return err
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"DeployItemsReconciled", "All DeployItems are successfully reconciled")
	o.exec.Status.ObservedGeneration = o.exec.Generation
	return o.UpdateStatus(ctx, phase, cond)
}

// listManagedDeployItems collects all deploy items that are managed by the execution.
// The managed execution is identified by the managed by label and the ownership.
func (o *Operation) listManagedDeployItems(ctx context.Context) ([]lsv1alpha1.DeployItem, error) {
	deployItemList := &lsv1alpha1.DeployItemList{}
	// todo: maybe use name and namespace
	if err := o.Client().List(ctx, deployItemList, client.MatchingLabels{lsv1alpha1.ExecutionManagedByLabel: o.exec.Name}, client.InNamespace(o.exec.Namespace)); err != nil {
		return nil, err
	}
	return deployItemList.Items, nil
}

// getExecutionItems creates an internal representation for all execution items.
// It also returns all removed deploy items that are not defined by the execution anymore.
func (o *Operation) getExecutionItems(items []lsv1alpha1.DeployItem) ([]executionItem, []lsv1alpha1.DeployItem) {
	execItems := make([]executionItem, len(o.exec.Spec.Executions))
	orphaned := items
	for i, exec := range o.exec.Spec.Executions {
		execItem := executionItem{
			Info: exec,
		}
		if j, ok := getDeployItemIndexByManagedName(items, exec.Name); ok {
			execItem.DeployItem = &items[j]
			copy(orphaned[i:], orphaned[j+1:])
			orphaned[len(orphaned)-1] = lsv1alpha1.DeployItem{}
			orphaned = orphaned[:len(orphaned)-1]
		}
	}
	return execItems, orphaned
}

// deployOrTrigger creates a new deploy item or triggers it if it already exists.
func (o *Operation) deployOrTrigger(ctx context.Context, item executionItem) error {

	if item.DeployItem == nil {
		item.DeployItem = &lsv1alpha1.DeployItem{}
		item.DeployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Info.Name)
		item.DeployItem.Namespace = o.exec.Namespace
	}

	if _, err := kubernetesutil.CreateOrUpdate(ctx, o.Client(), item.DeployItem, func() error {
		lsv1alpha1helper.SetOperation(&item.DeployItem.ObjectMeta, lsv1alpha1.ReconcileOperation)
		item.DeployItem.Spec.BlueprintRef = o.exec.Spec.BlueprintRef
		item.DeployItem.Spec.RegistryPullSecrets = o.exec.Spec.RegistryPullSecrets
		item.DeployItem.Spec.ImportReference = o.exec.Spec.ImportReference
		item.DeployItem.Spec.Type = item.Info.Type
		item.DeployItem.Spec.Configuration = item.Info.Configuration
		kubernetesutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedByLabel, o.exec.Name)
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
func (o *Operation) collectAndUpdateExports(ctx context.Context) error {
	var (
		values = make(map[string]interface{})
		err    error
	)
	for _, ref := range o.exec.Status.DeployItemReferences {
		deployItem := &lsv1alpha1.DeployItem{}
		if err := o.Client().Get(ctx, ref.Reference.NamespacedName(), deployItem); err != nil {
			return err
		}

		values, err = o.addExports(ctx, deployItem, values)
		if err != nil {
			return err
		}
	}

	return o.CreateOrUpdateExportReference(ctx, values)
}

// addExports loads the exports of a deploy item and adds it to the given values.
func (o *Operation) addExports(ctx context.Context, item *lsv1alpha1.DeployItem, values map[string]interface{}) (map[string]interface{}, error) {
	if item.Status.ExportReference == nil {
		return values, nil
	}
	secret := &corev1.Secret{}
	if err := o.Client().Get(ctx, item.Status.ExportReference.NamespacedName(), secret); err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(secret.Data[lsv1alpha1.DataObjectSecretDataKey], &data); err != nil {
		return nil, err
	}
	return utils.MergeMaps(values, data), nil
}
