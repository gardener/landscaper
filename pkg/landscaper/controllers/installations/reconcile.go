// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
)

func (c *Controller) reconcile(ctx context.Context, inst *lsv1alpha1.Installation) error {
	c.Log().Info("Reconcile installation", "name", inst.GetName(), "namespace", inst.GetNamespace())

	instOp, err := c.initPrerequisites(ctx, inst)
	if err != nil {
		return err
	}

	subinstallation := subinstallations.New(instOp)
	exec := executions.New(instOp)
	instOp.CurrentOperation = "Validate"

	execState, err := exec.CombinedState(ctx, instOp.Inst)
	if err != nil {
		return instOp.NewError(err, "CheckStatus", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	subState, err := subinstallation.CombinedState(ctx, instOp.Inst)
	if err != nil {
		return instOp.NewError(err, "CheckSubinstallationStatus", err.Error())
	}

	combinedState := lsv1alpha1helper.CombinedInstallationPhase(subState, lsv1alpha1.ComponentInstallationPhase(execState))

	// we have to wait until all children (subinstallations and execution) are finished
	if combinedState != "" && !lsv1alpha1helper.IsCompletedInstallationPhase(combinedState) {
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

	// check if the spec has changed
	eligibleToUpdate, err := c.eligibleToUpdate(ctx, instOp)
	if err != nil {
		return instOp.NewError(err, "EligibleForUpdate", err.Error())
	}
	if eligibleToUpdate {
		inst.Status.Phase = lsv1alpha1.ComponentPhasePending
		// need to return and not continue with export validation
		return c.Update(ctx, instOp)
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

	instOp.Inst.GetInfo().Status.Phase = lsv1alpha1.ComponentPhasePending
	if err := c.Update(ctx, instOp); err != nil {
		return err
	}

	delete(instOp.Inst.GetInfo().Annotations, lsv1alpha1.OperationAnnotation)
	if err := c.Client().Update(ctx, instOp.Inst.GetInfo()); err != nil {
		return err
	}

	instOp.Inst.GetInfo().Status.ObservedGeneration = instOp.Inst.GetInfo().Generation
	instOp.Inst.GetInfo().Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	return nil
}

// eligibleToUpdate checks whether the subinstallations and deploy items should be updated.
// The check succeeds if the installation's generation has changed or the imported deploy item versions have changed.
func (c *Controller) eligibleToUpdate(ctx context.Context, op *installations.Operation) (bool, error) {
	if op.Inst.GetInfo().Generation != op.Inst.GetInfo().Status.ObservedGeneration {
		return true, nil
	}

	validator := imports.NewValidator(op)
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
func (c *Controller) Update(ctx context.Context, op *installations.Operation) error {
	currOp := "Validate"
	inst := op.Inst
	if err := imports.NewValidator(op).ImportsSatisfied(ctx, inst); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "ImportsSatisfied", err.Error())
	}

	currOp = "Reconcile"
	// as all imports are satisfied we can collect and merge all imports
	// and then start the executions
	constructor := imports.NewConstructor(op)
	if err := constructor.Construct(ctx, inst); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "ConstructImports", err.Error())
	}

	if err := op.CreateOrUpdateImports(ctx); err != nil {
		inst.GetInfo().Status.LastError = lsv1alpha1helper.UpdatedError(inst.GetInfo().Status.LastError,
			"CreateImports",
			"unable to update import objects",
			err.Error())
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "CreateOrUpdateImports", err.Error())
	}

	inst.GetInfo().Status.Phase = lsv1alpha1.ComponentPhaseProgressing

	subinstallation := subinstallations.New(op)
	if err := subinstallation.Ensure(ctx); err != nil {
		return err
	}

	// todo: check if this can be moved to ensure
	if err := subinstallation.TriggerSubInstallations(ctx, inst.GetInfo(), lsv1alpha1.ReconcileOperation); err != nil {
		err = fmt.Errorf("unable to trigger subinstallations: %w", err)
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "ReconcileSubinstallations", err.Error())
	}

	exec := executions.New(op)
	if err := exec.Ensure(ctx, inst); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "ReconcileExecution", err.Error())
	}

	inst.GetInfo().Status.Imports = inst.ImportStatus().GetStatus()
	inst.GetInfo().Status.ObservedGeneration = inst.GetInfo().Generation
	inst.GetInfo().Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	inst.GetInfo().Status.LastError = nil
	return nil
}
