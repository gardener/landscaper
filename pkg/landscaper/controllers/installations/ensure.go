// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
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

func (a *actuator) Ensure(ctx context.Context, op *installations.Operation) error {
	// check that all referenced definitions have a corresponding installation
	inst := op.Inst
	subinstallation := subinstallations.New(op)
	exec := executions.New(op)

	execState, err := exec.CombinedState(ctx, inst)
	if err != nil {
		return err
	}

	subState, err := subinstallation.CombinedState(ctx, inst)
	if err != nil {
		return err
	}

	combinedState := lsv1alpha1helper.CombinedInstallationPhase(subState, lsv1alpha1.ComponentInstallationPhase(execState))

	// we have to wait until all children (subinstallations and execution) are finished
	if combinedState != "" && !lsv1alpha1helper.IsCompletedInstallationPhase(combinedState) {
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
		return nil
	}

	if lsv1alpha1helper.HasOperation(inst.Info.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: remove annotation
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseAborted
		if err := a.Client().Status().Update(ctx, inst.Info); err != nil {
			return err
		}
		return nil
	}

	// check if the spec has changed
	eligibleToUpdate, err := a.eligibleToUpdate(ctx, op, inst)
	if err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"EligibleForUpdate", "EligibleForUpdate", err.Error())
		return err
	}
	if eligibleToUpdate {
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhasePending
		if err := a.ApplyUpdate(ctx, op); err != nil {
			return err
		}

		inst.Info.Status.ObservedGeneration = inst.Info.Generation
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
		inst.Info.Status.LastError = nil
		// need to return and not continue with export validation
		return nil
	}

	if combinedState != lsv1alpha1.ComponentPhaseSucceeded {
		inst.Info.Status.Phase = combinedState
		return nil
	}

	dataExports, targetExports, err := exports.NewConstructor(op).Construct(ctx)
	if err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"ConstructExports",
			"error during export construction",
			err.Error())
		return fmt.Errorf("error during export construction: %w", err)
	}

	if err := op.CreateOrUpdateExports(ctx, dataExports, targetExports); err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"CreateExports",
			"unable to create exported dataobjects and targets",
			err.Error())
		return fmt.Errorf("unable to create exported dataobjects and targets: %w", err)
	}

	// update import status
	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

	// as all exports are validated, lets trigger dependant components
	// todo: check if this is a must, maybe track what we already successfully triggered
	// maybe we also need to increase the generation manually to signal a new config version
	if err := op.TriggerDependants(ctx); err != nil {
		err = fmt.Errorf("unable to trigger dependent installations: %w", err)
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"TriggerDependents",
			"",
			err.Error())
		return err
	}
	inst.Info.Status.LastError = nil
	return nil
}

//TODO: remove inst from signature
// eligibleToUpdate checks whether the subinstallations and deploy items should be updated.
// The check succeeds if the installation's generation has changed or the imported deploy item versions have changed.
func (a *actuator) eligibleToUpdate(ctx context.Context, op *installations.Operation, inst *installations.Installation) (bool, error) {
	if inst.Info.Generation != inst.Info.Status.ObservedGeneration {
		return true, nil
	}

	validator := imports.NewValidator(op)
	run, err := validator.OutdatedImports(ctx, inst)
	if err != nil {
		return run, err
	}
	if !run {
		return false, nil
	}

	return validator.CheckDependentInstallations(ctx, inst)
}

// ApplyUpdate redeploys subinstallations and deploy items.
func (a *actuator) ApplyUpdate(ctx context.Context, op *installations.Operation) error {
	inst := op.Inst
	if err := imports.NewValidator(op).ImportsSatisfied(ctx, inst); err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"ImportsSatisfied",
			"unable to check that all imports are satisfied",
			err.Error())
		return err
	}
	// as all imports are satisfied we can collect and merge all imports
	// and then start the executions
	constructor := imports.NewConstructor(op)
	importedValues, err := constructor.Construct(ctx, inst)
	if err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"CreateImports",
			"unable to collect imports",
			err.Error())
		return err
	}

	if err := op.CreateOrUpdateImports(ctx, importedValues); err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"CreateImports",
			"unable to update import objects",
			err.Error())
		return fmt.Errorf("unable to update import objects: %w", err)
	}

	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

	subinstallation := subinstallations.New(op)
	if err := subinstallation.Ensure(ctx, inst.Info, inst.Blueprint); err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"ReconcileSubinstallations",
			"unable to ensure sub installations",
			err.Error())
		return fmt.Errorf("unable to ensure sub installations: %w", err)
	}

	if err := subinstallation.TriggerSubInstallations(ctx, inst.Info, lsv1alpha1.ReconcileOperation); err != nil {
		return fmt.Errorf("unable to trigger subinstallations: %w", err)
	}

	exec := executions.New(op)
	if err := exec.Ensure(ctx, inst, importedValues); err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"ReconcileDeployItems",
			"unable to reconcile execution",
			err.Error())
		return fmt.Errorf("unable to ensure execution: %w", err)
	}
	inst.Info.Status.Imports = inst.ImportStatus().GetStatus()
	if err := a.Client().Status().Update(ctx, inst.Info); err != nil {
		inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Info.Status.LastError,
			"ApplyUpdate",
			"unable to update installation status",
			err.Error())
		return err
	}
	return nil
}
