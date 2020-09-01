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

package executions

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

func (o *ExecutionOperation) CombinedState(ctx context.Context, inst *installations.Installation) (lsv1alpha1.ExecutionPhase, error) {
	if inst.Info.Status.ExecutionReference == nil {
		return "", nil
	}

	exec := &lsv1alpha1.Execution{}
	if err := o.Client().Get(ctx, inst.Info.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		return "", err
	}
	return exec.Status.Phase, nil
}

func (o *ExecutionOperation) HandleUpdate(ctx context.Context, inst *installations.Installation) error {
	if inst.Info.Status.ExecutionReference == nil {
		return nil
	}

	exec := &lsv1alpha1.Execution{}
	if err := o.Client().Get(ctx, inst.Info.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		return err
	}

	if exec.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseFailed
		return nil
	}

	return nil
}

// GetExportedValues returns the exported values of the execution
func (o *ExecutionOperation) GetExportedValues(ctx context.Context, inst *installations.Installation) (*dataobjects.DataObject, error) {
	if inst.Info.Status.ExecutionReference == nil {
		return nil, nil
	}

	exec := &lsv1alpha1.Execution{}
	if err := o.Client().Get(ctx, inst.Info.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return o.GetExportForKey(ctx, exec, "")
}
