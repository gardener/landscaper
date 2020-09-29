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
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// cleanupOrphanedDeployItems deletes all orphaned deploy items that are not defined by their execution anymore.
func (o *Operation) cleanupOrphanedDeployItems(ctx context.Context, orphaned []lsv1alpha1.DeployItem) error {
	if len(orphaned) == 0 {
		return nil
	}
	for _, item := range orphaned {
		if item.DeletionTimestamp.IsZero() && o.checkGCDeletable(item, orphaned) {
			if err := o.Client().Delete(ctx, &item); err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("unable to delete deploy item %s", item.Name)
				}
			}
		}
	}
	return fmt.Errorf("waiting for %d orphaned deploy items to be deleted", len(orphaned))
}

// checkGCDeletable checks whether all deploy items depending on a given deploy item have been successfully deleted.
// only other deleted items are respected.
func (o *Operation) checkGCDeletable(item lsv1alpha1.DeployItem, items []lsv1alpha1.DeployItem) bool {
	for _, di := range items {
		if di.Name == item.Name {
			continue
		}

		dependsOn := sets.NewString(strings.Split(di.Annotations[lsv1alpha1.ExecutionDependsOnAnnotation], ",")...)
		if dependsOn.Has(item.Labels[lsv1alpha1.ExecutionManagedNameLabel]) {
			return false
		}
	}
	return true
}
