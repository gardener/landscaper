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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Delete handles the delete flow for a execution
func (o *Operation) Delete(ctx context.Context) error {
	// set state to deleting
	o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	managedItems, err := o.listManagedDeployItems(ctx)
	if err != nil {
		return fmt.Errorf("unable to list managed deploy items: %w", err)
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
					return fmt.Errorf("unable to delete deploy item %s of step %s: %w", item.DeployItem.Name, item.Info.Name, err)
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
	return o.Client().Update(ctx, o.exec)
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
