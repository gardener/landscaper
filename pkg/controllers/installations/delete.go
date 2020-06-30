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

package installations

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
)

func (a *actuator) ensureDeletion(ctx context.Context, op *installations.Operation, inst *installations.Installation) error {

	// check if suitable for deletion
	// - no sibling has imports that we export

	execDeleted, err := a.deleteExecution(ctx, inst)
	if err != nil {
		return err
	}

	subInstsDeleted, err := a.deleteSubInstallations(ctx, op, inst)
	if err != nil {
		return err
	}

	if !execDeleted || !subInstsDeleted {
		return errors.New("waiting for deletion")
	}

	controllerutil.RemoveFinalizer(inst.Info, lsv1alpha1.LandscaperFinalizer)
	return a.c.Update(ctx, inst.Info)
}

func (a *actuator) deleteExecution(ctx context.Context, inst *installations.Installation) (bool, error) {
	if inst.Info.Status.ExecutionReference == nil {
		return true, nil
	}
	exec := &lsv1alpha1.Execution{}
	if err := a.c.Get(ctx, inst.Info.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if !exec.DeletionTimestamp.IsZero() {
		if err := a.c.Delete(ctx, exec); err != nil {
			return false, err
		}
	}
	return false, nil
}

func (a *actuator) deleteSubInstallations(ctx context.Context, op *installations.Operation, inst *installations.Installation) (bool, error) {
	// todo: better error reporting as condition
	subInsts, err := subinstallations.New(op).GetSubInstallations(ctx, inst.Info)
	if err != nil {
		return false, err
	}
	if len(subInsts) == 0 {
		return true, nil
	}

	for _, inst := range subInsts {
		if !inst.DeletionTimestamp.IsZero() {
			if err := a.c.Delete(ctx, inst); err != nil {
				return false, err
			}
		}
	}

	return false, nil
}
