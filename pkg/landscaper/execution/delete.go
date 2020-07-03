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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

// Delete handles the delete flow for a execution
func (o *Operation) Delete(ctx context.Context) error {
	for i := len(o.exec.Spec.Executions) - 1; i >= 0; i-- {
		item := o.exec.Spec.Executions[i]
		ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, item.Name)
		if !ok {
			continue
		}

		deployItem := &lsv1alpha1.DeployItem{}
		if err := o.Client().Get(ctx, ref.Reference.NamespacedName(), deployItem); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}

		if deployItem.DeletionTimestamp.IsZero() {
			// todo: set operation to deleting item
			return o.Client().Delete(ctx, deployItem)
		}

		// update state
		if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
			if err := o.UpdateStatus(ctx, lsv1alpha1.ExecutionPhaseFailed); err != nil {
				return err
			}
		}

		return nil
	}

	controllerutil.RemoveFinalizer(o.exec, lsv1alpha1.LandscaperFinalizer)
	return o.Client().Update(ctx, o.exec)
}
