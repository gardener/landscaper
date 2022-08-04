// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

var (
	SiblingImportError = errors.New("a sibling still imports some of the exports")
)

func (c *Controller) handleDeletionPhaseInit(ctx context.Context, inst *lsv1alpha1.Installation) (fatalError lserrors.LsError, normalError lserrors.LsError) {
	op := "handleDeletionPhaseInit"

	if err := c.deleteAllowed(ctx, inst); err != nil {
		return nil, lserrors.NewWrappedError(err, op, "deleteAllowed", err.Error())
	}

	exec, err := executions.GetExecutionForInstallation(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "GetExecutionForInstallation", err.Error()), nil
	}

	if exec != nil {
		if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(inst.ObjectMeta) &&
			!lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(exec.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			if err := c.Writer().UpdateExecution(ctx, read_write_layer.W000102, exec); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "UpdateExecution", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "UpdateExecutionConflict", err.Error()), nil
			}
		}

		if exec.DeletionTimestamp.IsZero() {
			if err = c.Writer().DeleteExecution(ctx, read_write_layer.W000012, exec); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "DeleteExecutionConflict", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "DeleteExecution", err.Error()), nil
			}
		}
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListSubinstallations", err.Error()), nil
	}

	for _, subInst := range subInsts {
		if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(inst.ObjectMeta) &&
			!lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(subInst.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000103, subInst); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "UpdateInstallationConflict", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "UpdateInstallation", err.Error()), nil
			}
		}

		if subInst.DeletionTimestamp.IsZero() {
			if err = c.Writer().DeleteInstallation(ctx, read_write_layer.W000091, subInst); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "DeleteInstallationConflict", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "DeleteInstallation", err.Error()), nil
			}
		}
	}

	return nil, nil
}

func (c *Controller) handleDeletionPhaseTriggerDeleting(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	op := "handleDeletionPhaseTriggerDeleting"
	exec, err := executions.GetExecutionForInstallation(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "GetExecutionForInstallation", err.Error())
	}

	if exec != nil && exec.Status.JobID != inst.Status.JobID {
		exec.Status.JobID = inst.Status.JobID
		if err = c.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000093, exec); err != nil {
			return lserrors.NewWrappedError(err, op, "UpdateExecutionStatus", err.Error())
		}
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListSubinstallations", err.Error())
	}

	for _, subInst := range subInsts {
		if subInst.Status.JobID != inst.Status.JobID {
			subInst.Status.JobID = inst.Status.JobID
			if err = c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000094, subInst); err != nil {
				return lserrors.NewWrappedError(err, op, "UpdateInstallationStatus", err.Error())
			}
		}
	}

	return nil
}

func (c *Controller) handleDeletionPhaseDeleting(ctx context.Context, inst *lsv1alpha1.Installation) (allFinished bool, allDeleted bool, lsErr lserrors.LsError) {
	op := "handleDeletionPhaseDeleting"

	exec, err := executions.GetExecutionForInstallation(ctx, c.Client(), inst)
	if err != nil {
		return false, false, lserrors.NewWrappedError(err, op, "GetExecutionForInstallation", err.Error())
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return false, false, lserrors.NewWrappedError(err, op, "ListSubinstallations", err.Error())
	}

	if exec == nil && len(subInsts) == 0 {
		controllerutil.RemoveFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err = c.Writer().UpdateInstallation(ctx, read_write_layer.W000095, inst); err != nil {
			return false, false, lserrors.NewWrappedError(err, op, "UpdateInstallation", err.Error())
		}

		// touch siblings to speed up processing
		// a potential improvement is to only touch siblings exporting data for the current installation but this would
		// result in more complex coding and should only be done if the current approach results in performance problems
		_, siblings, err := installations.GetParentAndSiblings(ctx, c.Client(), inst)
		if err != nil {
			return false, false, lserrors.NewWrappedError(err, op, "GetParentAndSiblings", err.Error())
		}
		for _, nextSibling := range siblings {
			if !nextSibling.DeletionTimestamp.IsZero() {
				lsv1alpha1helper.Touch(&nextSibling.ObjectMeta)
				if err = c.Writer().UpdateInstallation(ctx, read_write_layer.W000147, nextSibling); err != nil {
					return false, false, lserrors.NewWrappedError(err, op, "TouchSibling", err.Error())
				}
			}
		}

		return true, true, nil
	}

	// check if all finished
	if exec != nil {
		if exec.Status.JobIDFinished != inst.Status.JobID {
			return false, false, nil
		}
	}

	for _, subInst := range subInsts {
		if subInst.Status.JobIDFinished != inst.Status.JobID {
			return false, false, nil
		}
	}

	// now we know that there exists subobjects and all of them are finished which means that they must have failed.
	return true, false, nil
}

func (c *Controller) handleDelete(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	var (
		currentOperation = "handleDelete"
		log              = logr.FromContextOrDiscard(ctx)
	)

	inst.Status.Phase = lsv1alpha1.ComponentPhaseDeleting
	inst.Status.ObservedGeneration = inst.GetGeneration()

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		if err := c.DeleteExecutionAndSubinstallations(ctx, inst); err != nil {
			return err
		}

		if lsErr := c.removeForceReconcileAnnotation(ctx, inst); lsErr != nil {
			return lsErr
		}

		return nil
	}

	if lsErr := c.deleteAllowed(ctx, inst); lsErr != nil {
		return lsErr
	}

	exec, subinst, err := readSubObjects(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ReadSubObjects", err.Error())
	}

	// We have to wait until all subobjects (execution & subinstallations) are completed which do not yet have a deletion
	// timestamp. We need not wait for objects with deletion timestamp. Actually, we should not wait for them in order
	// to avoid deadlocks like this: if subinstallation A with deletion timestamp has a dependency to subinstallation B
	// without deletion timestamp, then A does not complete before B is gone, and B would not get a deletion timestamp
	// before A is completed. This occurs if the process of adding the deletion timestamps was interrupted after A and
	// before B.
	if !allCompletedOrWithDeletionTimestamp(exec, subinst) {
		log.V(2).Info("Waiting for execution and subinstallations to be completed")
		return nil
	}

	err = c.DeleteExecutionAndSubinstallations(ctx, inst)
	return lserrors.NewErrorOrNil(err, currentOperation, "DeleteExecutionAndSubinstallations")
}

func (c *Controller) deleteAllowed(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	op := "DeleteInstallationAllowed"

	_, siblings, err := installations.GetParentAndSiblings(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err,
			op, "CalculateInstallationContext", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	// check if suitable for deletion
	// todo: replacements and internal deletions
	if checkIfSiblingImports(inst, installations.CreateInternalInstallationBases(siblings...)) {
		return lserrors.NewWrappedError(SiblingImportError,
			op, "SiblingImport", SiblingImportError.Error())
	}

	return nil
}

// DeleteExecutionAndSubinstallations deletes the execution and all subinstallations of the installation.
// The function does not wait for the successful deletion of all resources.
// It returns nil and should be called on every reconcile until it removes the finalizer form the current installation.
func (c *Controller) DeleteExecutionAndSubinstallations(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	op := "DeleteExecutionAndSubinstallations"

	writer := c.Writer()

	execDeleted, err := deleteExecution(ctx, writer, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "DeleteExecution", err.Error())
	}

	subInstsDeleted, err := deleteSubInstallations(ctx, writer, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "DeleteSubinstallations", err.Error())
	}

	if !execDeleted || !subInstsDeleted {
		if lsErr := c.removeReconcileAnnotation(ctx, inst); err != nil {
			return lsErr
		}
		return nil
	}

	controllerutil.RemoveFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
	err = writer.UpdateInstallation(ctx, read_write_layer.W000008, inst)
	return lserrors.NewErrorOrNil(err, op, "RemoveFinalizer")
}

func deleteExecution(ctx context.Context, kubeWriter *read_write_layer.Writer, kubeClient client.Client, inst *lsv1alpha1.Installation) (bool, error) {
	exec, err := executions.GetExecutionForInstallation(ctx, kubeClient, inst)
	if err != nil {
		return false, err
	}
	if exec == nil {
		return true, nil
	}

	if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(inst.ObjectMeta) {
		metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
		if err := kubeWriter.UpdateExecution(ctx, read_write_layer.W000024, exec); err != nil {
			return false, fmt.Errorf("unable to add delete-without-uninstall annotation to execution %s: %w",
				exec.Name, err)
		}
	}

	if exec.DeletionTimestamp.IsZero() {
		if err := kubeWriter.DeleteExecution(ctx, read_write_layer.W000035, exec); err != nil {
			return false, err
		}
	}

	// add reconcile or force reconcile annotation if present
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		lsv1alpha1helper.SetOperation(&exec.ObjectMeta, lsv1alpha1.ReconcileOperation)
		exec.Spec.ReconcileID = uuid.New().String()
		if err := kubeWriter.UpdateExecution(ctx, read_write_layer.W000078, exec); err != nil {
			return false, fmt.Errorf("unable to add reconcile label")
		}
	} else if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		lsv1alpha1helper.SetOperation(&exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
		if err := kubeWriter.UpdateExecution(ctx, read_write_layer.W000023, exec); err != nil {
			return false, fmt.Errorf("unable to add force reconcile label")
		}
	}
	return false, nil
}

func deleteSubInstallations(ctx context.Context, kubeWriter *read_write_layer.Writer, kubeClient client.Client, parentInst *lsv1alpha1.Installation) (bool, error) {
	subInsts, err := installations.ListSubinstallations(ctx, kubeClient, parentInst)
	if err != nil {
		return false, err
	}
	if len(subInsts) == 0 {
		return true, nil
	}

	if err := propagateDeleteWithoutUninstallAnnotation(ctx, kubeWriter, parentInst, subInsts); err != nil {
		return false, err
	}

	for _, subInst := range subInsts {
		if subInst.DeletionTimestamp.IsZero() {
			if err := kubeWriter.DeleteInstallation(ctx, read_write_layer.W000019, subInst); err != nil {
				return false, err
			}
		}

		if lsv1alpha1helper.HasOperation(parentInst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
			lsv1alpha1helper.SetOperation(&subInst.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
			if err := kubeWriter.UpdateInstallation(ctx, read_write_layer.W000005, subInst); err != nil {
				return false, fmt.Errorf("unable to add force reconcile annotation to subinstallation %s: %w", subInst.Name, err)
			}
		} else if lsv1alpha1helper.HasOperation(parentInst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
			lsv1alpha1helper.SetOperation(&subInst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			if err := kubeWriter.UpdateInstallation(ctx, read_write_layer.W000079, subInst); err != nil {
				return false, fmt.Errorf("unable to add reconcile annotation to subinstallation %s: %w", subInst.Name, err)
			}
		}
	}

	return false, nil
}

func propagateDeleteWithoutUninstallAnnotation(ctx context.Context, kubeWriter *read_write_layer.Writer, parentInst *lsv1alpha1.Installation, subInsts []*lsv1alpha1.Installation) error {
	op := "PropagateDeleteWithoutUninstallAnnotationToSubInstallation"

	if !lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(parentInst.ObjectMeta) {
		return nil
	}

	for _, subInst := range subInsts {
		metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
		if err := kubeWriter.UpdateInstallation(ctx, read_write_layer.W000006, subInst); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			msg := fmt.Sprintf("unable to update subinstallation %s: %s", subInst.Name, err.Error())
			return lserrors.NewWrappedError(err, op, "Update", msg)
		}
	}

	return nil
}

// checkIfSiblingImports checks if a sibling imports any of the installations exports.
func checkIfSiblingImports(inst *lsv1alpha1.Installation, siblings []*installations.InstallationBase) bool {
	for _, sibling := range siblings {
		for _, dataImport := range inst.Spec.Exports.Data {
			if sibling.IsImportingData(dataImport.DataRef) {
				return true
			}
		}
		for _, targetImport := range inst.Spec.Exports.Targets {
			if sibling.IsImportingData(targetImport.Target) {
				return true
			}
		}
	}
	return false
}

func readSubObjects(ctx context.Context, cl client.Client, inst *lsv1alpha1.Installation) (
	*lsv1alpha1.Execution, []*lsv1alpha1.Installation, error) {

	exec, err := executions.GetExecutionForInstallation(ctx, cl, inst)
	if err != nil {
		return nil, nil, err
	}

	subinsts, err := installations.ListSubinstallations(ctx, cl, inst)
	if err != nil {
		return nil, nil, err
	}

	return exec, subinsts, nil
}

func allCompletedOrWithDeletionTimestamp(exec *lsv1alpha1.Execution, subinsts []*lsv1alpha1.Installation) bool {
	filterExec := func(exec *lsv1alpha1.Execution) bool {
		return exec == nil ||
			!exec.DeletionTimestamp.IsZero() ||
			(exec.Generation == exec.Status.ObservedGeneration && lsv1alpha1helper.IsCompletedInstallationPhase(lsv1alpha1.ComponentInstallationPhase(exec.Status.Phase)))
	}

	filterInst := func(inst *lsv1alpha1.Installation) bool {
		return inst == nil ||
			!inst.DeletionTimestamp.IsZero() ||
			(inst.Generation == inst.Status.ObservedGeneration && lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase))
	}

	result := filterExec(exec)
	for _, subinst := range subinsts {
		result = result && filterInst(subinst)
	}

	return result
}
