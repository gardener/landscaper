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
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
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
		log.V(2).Info("Waiting for all deploy items and subinstallations to be completed")
		inst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
		return nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: remove annotation
		inst.Status.Phase = lsv1alpha1.ComponentPhaseAborted
		if err := c.Client().Status().Update(ctx, inst); err != nil {
			return err
		}
		return nil
	}

	instOp, err := c.initPrerequisites(ctx, inst)
	if err != nil {
		return err
	}
	instOp.CurrentOperation = currentOperation

	// check if the spec has changed
	eligibleToUpdate, err := c.eligibleToUpdate(ctx, instOp)
	if err != nil {
		return instOp.NewError(err, "EligibleForUpdate", err.Error())
	}
	if eligibleToUpdate {
		inst.Status.Phase = lsv1alpha1.ComponentPhasePending
		// need to return and not continue with export validation
		return c.Update(ctx, instOp, false)
	}

	if combinedState != lsv1alpha1.ComponentPhaseSucceeded {
		inst.Status.Phase = combinedState
		return nil
	}

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
	if err := instOp.TriggerDependants(ctx); err != nil {
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
	if err := c.Update(ctx, instOp, true); err != nil {
		return err
	}

	delete(instOp.Inst.Info.Annotations, lsv1alpha1.OperationAnnotation)
	if err := c.Client().Update(ctx, instOp.Inst.Info); err != nil {
		return err
	}

	instOp.Inst.Info.Status.ObservedGeneration = instOp.Inst.Info.Generation
	instOp.Inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	return nil
}

// eligibleToUpdate checks whether the subinstallations and deploy items should be updated.
// The check succeeds if the installation's generation has changed or the imported deploy item versions have changed.
func (c *Controller) eligibleToUpdate(ctx context.Context, op *installations.Operation) (bool, error) {
	if op.Inst.Info.Generation != op.Inst.Info.Status.ObservedGeneration {
		return true, nil
	}

	validator, err := imports.NewValidator(ctx, op)
	if err != nil {
		return false, err
	}
	run, err := validator.OutdatedImports(ctx)
	if err != nil {
		return run, err
	}
	if !run {
		return false, nil
	}

	return validator.CheckDependentInstallations(ctx)
}

// Update redeploys subinstallations and deploy items.
func (c *Controller) Update(ctx context.Context, op *installations.Operation, forced bool) error {
	currOp := "Validate"
	inst := op.Inst

	val, err := imports.NewValidator(ctx, op)
	if err != nil {
		err = fmt.Errorf("unable to init import validator: %w", err)
		return lserrors.NewWrappedError(err,
			currOp, "ImportsSatisfied", err.Error())
	}
	if err := val.ImportsSatisfied(ctx, inst); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ImportsSatisfied", err.Error())
	}

	currOp = "Reconcile"
	// as all imports are satisfied we can collect and merge all imports
	// and then start the executions
	constructor := imports.NewConstructor(op)
	if err := constructor.Construct(ctx, inst); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ConstructImports", err.Error())
	}

	if err := op.CreateOrUpdateImports(ctx); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "CreateOrUpdateImports", err.Error())
	}

	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

	subinstallation := subinstallations.New(op)
	subinstallation.Forced = forced
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
