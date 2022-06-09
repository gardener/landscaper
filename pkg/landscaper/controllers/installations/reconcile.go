// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (c *Controller) reconcile(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	var (
		currentOperation = "Validate"
		log              = logr.FromContextOrDiscard(ctx)
	)
	log.Info("Reconcile installation", "name", inst.GetName(), "namespace", inst.GetNamespace())

	combinedState, lsErr := c.combinedPhaseOfSubobjects(ctx, inst, currentOperation)
	if lsErr != nil {
		inst.Status.Phase = lsv1alpha1.ComponentPhaseInit
		return lsErr
	}

	if !lsv1alpha1helper.IsCompletedInstallationPhase(combinedState) {
		log.V(2).Info("Waiting for all deploy items and nested installations to be completed")
		inst.Status.Phase = lsv1alpha1.ComponentPhaseInit
		return nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: remove annotation
		inst.Status.Phase = lsv1alpha1.ComponentPhaseAborted
		if err := c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000014, inst); err != nil {
			return lserrors.NewWrappedError(err, currentOperation, "SetInstallationPhaseAborted", err.Error())
		}
		return nil
	}

	instOp, lsErr := c.initPrerequisites(ctx, inst)
	if lsErr != nil {
		inst.Status.Phase = lsv1alpha1.ComponentPhaseInit
		return lsErr
	}
	instOp.CurrentOperation = currentOperation

	rh, err := reconcilehelper.NewReconcileHelper(ctx, instOp)
	if err != nil {
		inst.Status.Phase = lsv1alpha1.ComponentPhaseInit
		return lserrors.NewWrappedError(err, currentOperation, "NewReconcileHelper", err.Error())
	}

	// check if a reconcile is required due to changed spec or imports
	updateRequired, err := rh.UpdateRequired()
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "IsUpdateRequired", err.Error())
	}
	if updateRequired {
		inst.Status.Phase = lsv1alpha1.ComponentPhasePending

		dependedOnSiblings, err := rh.FetchDependencies()
		if err != nil {
			return lserrors.NewWrappedError(err, currentOperation, "FetchDependencies", err.Error())
		}

		// check whether the installation can be updated
		err = rh.UpdateAllowed(dependedOnSiblings)
		if err != nil {
			return lserrors.NewWrappedError(err, currentOperation, "IsUpdateAllowed", err.Error())
		}

		imps, err := rh.GetImports()
		if err != nil {
			return lserrors.NewWrappedError(err, currentOperation, "GetImports", err.Error())
		}

		return c.Update(ctx, instOp, imps)
	}

	if lsErr := c.removeReconcileAnnotation(ctx, instOp.Inst.Info); lsErr != nil {
		return lsErr
	}

	if combinedState != lsv1alpha1.ComponentPhaseSucceeded {
		inst.Status.Phase = combinedState
		return nil
	}

	// no update required, continue with exports
	// construct imports so that they are available for export templating
	imps, err := rh.GetImports()
	if err != nil {
		return instOp.NewError(err, "GetImportsForExports", err.Error())
	}
	impCon := imports.NewConstructor(instOp)
	err = impCon.Construct(ctx, imps)
	if err != nil {
		return instOp.NewError(err, "ConstructImportsForExports", err.Error())
	}
	instOp.CurrentOperation = "Completing"
	dataExports, targetExports, err := exports.NewConstructor(instOp).Construct(ctx)
	if err != nil {
		return instOp.NewError(err, "ConstructExports", err.Error())
	}

	if err := instOp.CreateOrUpdateExports(ctx, dataExports, targetExports); err != nil {
		return instOp.NewError(err, "CreateOrUpdateExports", err.Error())
	}

	// as all exports are validated, lets trigger dependant components
	// todo: check if this is a must, maybe track what we already successfully triggered
	// maybe we also need to increase the generation manually to signal a new config version
	if err := instOp.TriggerDependents(ctx); err != nil {
		err = fmt.Errorf("unable to trigger dependent installations: %w", err)
		return instOp.NewError(err, "TriggerDependents", err.Error())
	}

	// update import status
	inst.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

	return nil
}

func (c *Controller) combinedPhaseOfSubobjects(ctx context.Context, inst *lsv1alpha1.Installation,
	currentOperation string) (lsv1alpha1.ComponentInstallationPhase, lserrors.LsError) {
	execState, err := executions.CombinedPhase(ctx, c.Client(), inst)
	if err != nil {
		return "", lserrors.NewWrappedError(err,
			currentOperation, "CheckExecutionStatus", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	subState, err := subinstallations.CombinedPhase(ctx, c.Client(), inst)
	if err != nil {
		return "", lserrors.NewWrappedError(err,
			currentOperation, "CheckSubinstallationStatus", err.Error())
	}

	combinedState := lsv1alpha1helper.CombinedInstallationPhase(subState, lsv1alpha1.ComponentInstallationPhase(execState))

	// we have to wait until all children (subinstallations and execution) are finished
	if combinedState == "" {
		// If combinedState is empty, this means there are neither subinstallations nor executions
		// and an 'empty' installation is Succeeded by default
		combinedState = lsv1alpha1.ComponentPhaseSucceeded
	}

	return combinedState, nil
}

func (c *Controller) forceReconcile(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	currentOperation := "ForceReconcile"
	c.Log().Info("Force Reconcile installation", "name", inst.GetName(), "namespace", inst.GetNamespace())
	instOp, lsErr := c.initPrerequisites(ctx, inst)
	if lsErr != nil {
		return lsErr
	}

	instOp.Inst.Info.Status.Phase = lsv1alpha1.ComponentPhasePending
	rh, err := reconcilehelper.NewReconcileHelper(ctx, instOp)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "NewReconcileHelper", err.Error())
	}

	// it is only checked whether the imports are satisfied,
	// the check whether installations this one depends on are succeeded is skipped
	if err := rh.ImportsSatisfied(); err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ImportsSatisfied", err.Error())
	}

	imps, err := rh.GetImports()
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "GetImports", err.Error())
	}

	if err := c.Update(ctx, instOp, imps); err != nil {
		return err
	}

	if lsErr := c.removeForceReconcileAnnotation(ctx, instOp.Inst.Info); lsErr != nil {
		return lsErr
	}

	instOp.Inst.Info.Status.ObservedGeneration = instOp.Inst.Info.Generation
	instOp.Inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	return nil
}

// Update redeploys subinstallations and deploy items.
func (c *Controller) Update(ctx context.Context, op *installations.Operation, imps *imports.Imports) lserrors.LsError {
	inst := op.Inst
	currOp := "Reconcile"
	// collect and merge all imports and start the Executions
	constructor := imports.NewConstructor(op)
	if err := constructor.Construct(ctx, imps); err != nil {
		return lserrors.NewWrappedError(err, currOp, "ConstructImports", err.Error())
	}

	if err := op.CreateOrUpdateImports(ctx); err != nil {
		return lserrors.NewWrappedError(err, currOp, "CreateOrUpdateImports", err.Error())
	}

	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

	subinstallation := subinstallations.New(op)
	if err := subinstallation.Ensure(ctx); err != nil {
		return lserrors.NewWrappedError(err, currOp, "EnsureSubinstallations", err.Error())
	}

	// todo: check if this can be moved to ensure
	if err := subinstallation.TriggerSubInstallations(ctx, inst.Info); err != nil {
		err = fmt.Errorf("unable to trigger subinstallations: %w", err)
		return lserrors.NewWrappedError(err, currOp, "ReconcileSubinstallations", err.Error())
	}

	exec := executions.New(op)
	if err := exec.Ensure(ctx, inst); err != nil {
		return lserrors.NewWrappedError(err, currOp, "ReconcileExecution", err.Error())
	}

	if lsErr := c.removeReconcileAnnotation(ctx, inst.Info); lsErr != nil {
		return lsErr
	}

	inst.Info.Status.Imports = inst.ImportStatus().GetStatus()
	inst.Info.Status.ObservedGeneration = inst.Info.Generation
	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	return nil
}

func (c *Controller) removeReconcileAnnotation(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		logr.FromContextOrDiscard(ctx).V(7).Info("remove reconcile annotation")
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000009, inst); client.IgnoreNotFound(err) != nil {
			return lserrors.NewWrappedError(err, "RemoveReconcileAnnotation", "UpdateInstallation", err.Error())
		}
	}
	return nil
}

func (c *Controller) removeForceReconcileAnnotation(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		logr.FromContextOrDiscard(ctx).V(7).Info("remove force reconcile annotation")
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000003, inst); err != nil {
			return lserrors.NewWrappedError(err, "RemoveForceReconcileAnnotation", "UpdateInstallation", err.Error())
		}
	}
	return nil
}
