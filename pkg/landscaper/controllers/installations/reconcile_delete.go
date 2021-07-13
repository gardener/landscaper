// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
)

var (
	SiblingImportError      = errors.New("a sibling still imports some of the exports")
	WaitingForDeletionError = errors.New("waiting for deletion")
)

func (c *Controller) handleDelete(ctx context.Context, inst *lsv1alpha1.Installation) error {
	instOp, err := c.initPrerequisites(ctx, inst)
	if err != nil {
		return err
	}
	return EnsureDeletion(ctx, instOp)
}

func EnsureDeletion(ctx context.Context, op *installations.Operation) error {
	op.CurrentOperation = "Deletion"
	op.Inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseDeleting
	// check if suitable for deletion
	// todo: replacements and internal deletions
	if checkIfSiblingImports(op) {
		return SiblingImportError
	}

	execDeleted, err := deleteExecution(ctx, op)
	if err != nil {
		return op.NewError(err, "DeleteExecution", err.Error())
	}

	subInstsDeleted, err := deleteSubInstallations(ctx, op)
	if err != nil {
		return op.NewError(err, "DeleteSubinstallations", err.Error())
	}

	// remove force reconcile annotation
	if metav1.HasAnnotation(op.Inst.Info.ObjectMeta, lsv1alpha1.OperationAnnotation) {
		delete(op.Inst.Info.Annotations, lsv1alpha1.OperationAnnotation)
		if err := op.Client().Update(ctx, op.Inst.Info); err != nil {
			return op.NewError(err, "RemoveOperationAnnotation", "Unable to remove operation annotation")
		}
	}

	if !execDeleted || !subInstsDeleted {
		return WaitingForDeletionError
	}

	controllerutil.RemoveFinalizer(op.Inst.Info, lsv1alpha1.LandscaperFinalizer)
	return op.Client().Update(ctx, op.Inst.Info)
}

func deleteExecution(ctx context.Context, op *installations.Operation) (bool, error) {
	exec := &lsv1alpha1.Execution{}
	if err := op.Client().Get(ctx, kutil.ObjectKeyFromObject(op.Inst.Info), exec); err != nil {
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

	// add force reconcile annotation if present
	if lsv1alpha1helper.HasOperation(op.Inst.Info.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		lsv1alpha1helper.SetOperation(&exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
		if err := op.Client().Update(ctx, exec); err != nil {
			return false, fmt.Errorf("unable to add force reconcile label")
		}
	}
	return false, nil
}

func deleteSubInstallations(ctx context.Context, op *installations.Operation) (bool, error) {
	op.CurrentOperation = "DeleteSubInstallation"
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
		if lsv1alpha1helper.HasOperation(op.Inst.Info.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
			if err := op.Client().Update(ctx, inst); err != nil {
				return false, fmt.Errorf("unable to add force reconcile annotation to subinstallation %s: %w", inst.Name, err)
			}
		}
	}

	return false, nil
}

func checkIfSiblingImports(op *installations.Operation) bool {
	for _, sibling := range op.Context().Siblings {
		for _, dataImports := range op.Inst.Info.Spec.Exports.Data {
			if sibling.IsImportingData(dataImports.DataRef) {
				return true
			}
		}
		for _, targetImport := range op.Inst.Info.Spec.Exports.Targets {
			if sibling.IsImportingData(targetImport.Target) {
				return true
			}
		}
	}
	return false
}
