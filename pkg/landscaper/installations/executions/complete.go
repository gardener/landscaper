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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

func (o *ExecutionOperation) Completed(ctx context.Context, inst *installations.Installation) (bool, error) {
	if inst.Info.Status.ExecutionReference == nil {
		return true, nil
	}

	exec := &lsv1alpha1.Execution{}
	if err := o.Client().Get(ctx, inst.Info.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		return false, err
	}
	return exec.Status.Phase == lsv1alpha1.ExecutionPhaseCompleted || exec.Status.Phase == lsv1alpha1.ExecutionPhaseFailed, nil
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