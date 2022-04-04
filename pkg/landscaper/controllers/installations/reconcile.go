// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

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

func (c *Controller) reconcile(ctx context.Context, inst *lsv1alpha1.Installation) error {
	var (
		currentOperation = "Validate"
		log              = logr.FromContextOrDiscard(ctx)
	)
	log.Info("Reconcile installation", "name", inst.GetName(), "namespace", inst.GetNamespace())

	execState, err := executions.CombinedPhase(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currentOperation, "CheckExecutionStatus", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	subState, err := subinstallations.CombinedPhase(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currentOperation, "CheckSubinstallationStatus", err.Error())
	}

	combinedState := lsv1alpha1helper.CombinedInstallationPhase(subState, lsv1alpha1.ComponentInstallationPhase(execState))

	// we have to wait until all children (subinstallations and execution) are finished
	if combinedState == "" {
		// If combinedState is empty, this means there are neither subinstallations nor executions
		// and an 'empty' installation is Succeeded by default
		combinedState = lsv1alpha1.ComponentPhaseSucceeded
	} else if !lsv1alpha1helper.IsCompletedInstallationPhase(combinedState) {
		log.V(2).Info("Waiting for all deploy items and nested installations to be completed")
		inst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
		return nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: remove annotation
		inst.Status.Phase = lsv1alpha1.ComponentPhaseAborted
		if err := read_write_layer.UpdateInstallationStatus(ctx, read_write_layer.W000014, c.Client().Status(), inst); err != nil {
			return err
		}
		return nil
	}

	instOp, err := c.initPrerequisites(ctx, inst)
	if err != nil {
		return err
	}
	instOp.CurrentOperation = currentOperation

	rh := reconcilehelper.NewReconcileHelper(ctx, instOp)

	// check if a reconcile is required due to changed spec or imports
	updateRequired, err := rh.UpdateRequired()
	if err != nil {
		return err
	}
	if updateRequired {
		inst.Status.Phase = lsv1alpha1.ComponentPhasePending
		// check whether the installation can be updated
		err := rh.UpdateAllowed()
		if err != nil {
			return err
		}

		imps, err := rh.GetImports()
		if err != nil {
			return err
		}
		return c.Update(ctx, instOp, imps)
	}

	if combinedState != lsv1alpha1.ComponentPhaseSucceeded {
		inst.Status.Phase = combinedState
		return nil
	}

	// no update required, continue with exports
	instOp.CurrentOperation = "Completing"
	dataExports, targetExports, err := exports.NewConstructor(instOp).Construct(ctx)
	if err != nil {
		return instOp.NewError(err, "ConstructExports", err.Error())
	}

	if err := instOp.CreateOrUpdateExports(ctx, dataExports, targetExports); err != nil {
		return instOp.NewError(err, "CreateOrUpdateExports", err.Error())
	}

	// update import status
	inst.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

	// as all exports are validated, lets trigger dependant components
	// todo: check if this is c must, maybe track what we already successfully triggered
	// maybe we also need to increase the generation manually to signal c new config version
	if err := instOp.TriggerDependents(ctx); err != nil {
		err = fmt.Errorf("unable to trigger dependent installations: %w", err)
		return instOp.NewError(err, "TriggerDependents", err.Error())
	}
	inst.Status.LastError = nil
	return nil
}

func (c *Controller) forceReconcile(ctx context.Context, inst *lsv1alpha1.Installation) error {
	c.Log().Info("Force Reconcile installation", "name", inst.GetName(), "namespace", inst.GetNamespace())
	instOp, err := c.initPrerequisites(ctx, inst)
	if err != nil {
		return err
	}

	instOp.Inst.Info.Status.Phase = lsv1alpha1.ComponentPhasePending
	rh := reconcilehelper.NewReconcileHelper(ctx, instOp)
	// it is only checked whether the imports are satisfied,
	// the check whether installations this one depends on are succeeded is skipped
	if err := rh.ImportsSatisfied(); err != nil {
		return err
	}
	imps, err := rh.GetImports()
	if err != nil {
		return err
	}
	if err := c.Update(ctx, instOp, imps); err != nil {
		return err
	}

	delete(instOp.Inst.Info.Annotations, lsv1alpha1.OperationAnnotation)
	if err := read_write_layer.UpdateInstallation(ctx, read_write_layer.W000012, c.Client(), instOp.Inst.Info); err != nil {
		return err
	}

	instOp.Inst.Info.Status.ObservedGeneration = instOp.Inst.Info.Generation
	instOp.Inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	return nil
}

// Update redeploys subinstallations and deploy items.
func (c *Controller) Update(ctx context.Context, op *installations.Operation, imps *imports.Imports) error {
	inst := op.Inst
	currOp := "Reconcile"
	// collect and merge all imports and start the Executions
	constructor := imports.NewConstructor(op)
	if err := constructor.Construct(ctx, imps); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ConstructImports", err.Error())
	}

	if err := op.CreateOrUpdateImports(ctx); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "CreateOrUpdateImports", err.Error())
	}

	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

	subinstallation := subinstallations.New(op)
	if err := subinstallation.Ensure(ctx); err != nil {
		return err
	}

	// todo: check if this can be moved to ensure
	if err := subinstallation.TriggerSubInstallations(ctx, inst.Info, lsv1alpha1.ReconcileOperation); err != nil {
		err = fmt.Errorf("unable to trigger subinstallations: %w", err)
		return lserrors.NewWrappedError(err,
			currOp, "ReconcileSubinstallations", err.Error())
	}

	exec := executions.New(op)
	if err := exec.Ensure(ctx, inst); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ReconcileExecution", err.Error())
	}

	inst.Info.Status.Imports = inst.ImportStatus().GetStatus()
	inst.Info.Status.ObservedGeneration = inst.Info.Generation
	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	inst.Info.Status.LastError = nil
	return nil
}
