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

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

var (
	SiblingImportError = errors.New("a sibling still imports some of the exports")
)

func (c *Controller) handleDelete(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	var (
		currentOperation = "Deletion"
		log              = logr.FromContextOrDiscard(ctx)
	)

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		if err := DeleteExecutionAndSubinstallations(ctx, c.Writer(), c.Client(), inst); err != nil {
			return err
		}

		log.V(7).Info("remove force reconcile annotation")
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000003, inst); err != nil {
			return lserrors.NewWrappedError(err,
				currentOperation, "RemoveOperationAnnotation", "Unable to remove operation annotation")
		}

		return nil
	}

	_, siblings, err := installations.GetParentAndSiblings(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currentOperation, "CalculateInstallationContext", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	// check if suitable for deletion
	// todo: replacements and internal deletions
	if checkIfSiblingImports(inst, installations.CreateInternalInstallationBases(siblings...)) {
		return lserrors.NewWrappedError(SiblingImportError,
			currentOperation, "SiblingImport", SiblingImportError.Error())
	}

	execPhase, err := executions.CombinedPhase(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currentOperation, "CheckExecutionStatus", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	subPhase, err := subinstallations.CombinedPhase(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "CheckSubinstallationStatus", err.Error())
	}

	// if no installations nor an execution is deployed both phases are empty. Then we can simply skip the deletion.
	if (len(execPhase) + len(subPhase)) == 0 {
		controllerutil.RemoveFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000004, inst)
		return lserrors.NewErrorOrNil(err, currentOperation, "RemoveFinalizer")
	}

	combinedState := lsv1alpha1helper.CombinedInstallationPhase(subPhase, lsv1alpha1.ComponentInstallationPhase(execPhase))

	// we have to wait until all children (subinstallations and execution) are finished
	if combinedState != "" && !lsv1alpha1helper.IsCompletedInstallationPhase(combinedState) {
		log.V(2).Info("Waiting for all deploy items and subinstallations to be completed")
		inst.Status.Phase = lsv1alpha1.ComponentPhaseDeleting
		return nil
	}

	err = DeleteExecutionAndSubinstallations(ctx, c.Writer(), c.Client(), inst)
	return lserrors.NewErrorOrNil(err, currentOperation, "DeleteExecutionAndSubinstallations")
}

// DeleteExecutionAndSubinstallations deletes the execution and all subinstallations of the installation.
// The function does not wait for the successful deletion of all resources.
// It returns nil and should be called on every reconcile until it removes the finalizer form the current installation.
func DeleteExecutionAndSubinstallations(ctx context.Context, writer *read_write_layer.Writer, c client.Client, inst *lsv1alpha1.Installation) lserrors.LsError {
	op := "Deletion"
	inst.Status.Phase = lsv1alpha1.ComponentPhaseDeleting

	execDeleted, err := deleteExecution(ctx, writer, c, inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "DeleteExecution", err.Error())
	}

	subInstsDeleted, err := deleteSubInstallations(ctx, writer, c, inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "DeleteSubinstallations", err.Error())
	}

	if !execDeleted || !subInstsDeleted {
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

	// add force reconcile annotation if present
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
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
		for _, dataImports := range inst.Spec.Exports.Data {
			if sibling.IsImportingData(dataImports.DataRef) {
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
