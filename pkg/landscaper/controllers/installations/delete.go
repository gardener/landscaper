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

func EnsureDeletion(ctx context.Context, op *installations.Operation) error {
	// check if suitable for deletion
	// todo: replacements and internal deletions
	if checkIfSiblingImports(op) {
		return errors.New("a sibling still imports some of the exports")
	}

	execDeleted, err := deleteExecution(ctx, op)
	if err != nil {
		return err
	}

	subInstsDeleted, err := deleteSubInstallations(ctx, op)
	if err != nil {
		return err
	}

	if !execDeleted || !subInstsDeleted {
		return errors.New("waiting for deletion")
	}

	controllerutil.RemoveFinalizer(op.Inst.Info, lsv1alpha1.LandscaperFinalizer)
	return op.Client().Update(ctx, op.Inst.Info)
}

func deleteExecution(ctx context.Context, op *installations.Operation) (bool, error) {
	if op.Inst.Info.Status.ExecutionReference == nil {
		return true, nil
	}
	exec := &lsv1alpha1.Execution{}
	if err := op.Client().Get(ctx, op.Inst.Info.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if exec.DeletionTimestamp.IsZero() {
		if err := op.Client().Delete(ctx, exec); err != nil {
			return false, err
		}
	}
	return false, nil
}

func deleteSubInstallations(ctx context.Context, op *installations.Operation) (bool, error) {
	// todo: better error reporting as condition
	subInsts, err := subinstallations.New(op).GetSubInstallations(ctx, op.Inst.Info)
	if err != nil {
		return false, err
	}
	if len(subInsts) == 0 {
		return true, nil
	}

	for _, inst := range subInsts {
		if inst.DeletionTimestamp.IsZero() {
			if err := op.Client().Delete(ctx, inst); err != nil {
				return false, err
			}
		}
	}

	return false, nil
}

func checkIfSiblingImports(op *installations.Operation) bool {
	for _, sibling := range op.Context().Siblings {
		for _, exportMapping := range op.Inst.Info.Spec.Exports {
			if _, err := sibling.GetImportMappingFrom(exportMapping.To); err == nil {
				return true
			}
		}
	}
	return false
}
