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

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils"
)

// Reconcile contains the reconcile logic for a execution item that schedules multiple DeployItems.
func (o *Operation) Reconcile(ctx context.Context) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.exec.Status.Conditions, lsv1alpha1.ReconcileDeployItemsCondition)

	// todo: make it possible to specify a dag
	var (
		phase lsv1alpha1.ExecutionPhase
	)
	for _, item := range o.exec.Spec.Executions {
		if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, item.Name); ok {
			deployItem := &lsv1alpha1.DeployItem{}
			if err := o.Client().Get(ctx, ref.Reference.NamespacedName(), deployItem); err != nil {
				return err
			}
			phase = lsv1alpha1helper.CombinedExecutionPhase(phase, deployItem.Status.Phase)

			if !lsv1alpha1helper.IsCompletedExecutionPhase(deployItem.Status.Phase) {
				return nil
			}

			if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
				cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"DeployItemFailed", fmt.Sprintf("DeployItem %s (%s) is in failed state", item.Name, ref.Reference.NamespacedName().String()))
				return o.UpdateStatus(ctx, lsv1alpha1.ExecutionPhaseFailed, cond)
			}

			// we already updated this item
			if ref.Reference.ObservedGeneration == o.exec.Generation {
				continue
			}
		}

		return o.deployOrTrigger(ctx, item)
	}

	if err := o.collectAndUpdateExports(ctx); err != nil {
		return err
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"DeployItemsReconciled", "All DeployItems are successfully reconciled")
	o.exec.Status.ObservedGeneration = o.exec.Generation
	return o.UpdateStatus(ctx, phase, cond)
}

// deployOrTrigger creates a new deploy item or triggers it if it already exists.
func (o *Operation) deployOrTrigger(ctx context.Context, item lsv1alpha1.ExecutionItem) error {
	deployItem := &lsv1alpha1.DeployItem{}
	deployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Name)
	deployItem.Namespace = o.exec.Namespace
	if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, item.Name); ok {
		deployItem.Name = ref.Reference.Name
		deployItem.Namespace = ref.Reference.Namespace
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), deployItem, func() error {
		lsv1alpha1helper.SetOperation(&deployItem.ObjectMeta, lsv1alpha1.ReconcileOperation)
		deployItem.Spec.Type = item.Type
		deployItem.Spec.Configuration = item.Configuration
		deployItem.Spec.ImportReference = o.exec.Spec.ImportReference
		return controllerutil.SetOwnerReference(o.exec, deployItem, o.Scheme())
	}); err != nil {
		return err
	}

	ref := lsv1alpha1.VersionedNamedObjectReference{}
	ref.Name = item.Name
	ref.Reference.Name = deployItem.Name
	ref.Reference.Namespace = deployItem.Namespace
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
	do, err := o.GetDataObjectFromSecret(ctx, item.Status.ExportReference.NamespacedName())
	if err != nil {
		return nil, err
	}
	return utils.MergeMaps(values, do.Data), nil
}
