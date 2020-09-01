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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
)

func (a *actuator) Ensure(ctx context.Context, op *installations.Operation, inst *installations.Installation) error {
	// check that all referenced definitions have a corresponding installation
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
	// todo: or if our imports have changed since last reconcile => check if config hash has changed
	if inst.Info.Generation != inst.Info.Status.ObservedGeneration {
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhasePending
		if err := a.StartNewReconcile(ctx, op, inst); err != nil {
			return err
		}

		inst.Info.Status.ObservedGeneration = inst.Info.Generation
		inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

		// need to return and not continue with export validation
		return nil
	}

	if combinedState != lsv1alpha1.ComponentPhaseSucceeded {
		inst.Info.Status.Phase = combinedState
		return nil
	}

	exportedValues, err := exports.NewConstructor(op).Construct(ctx, inst)
	if err != nil {
		a.Log().Error(err, "error during export construction")
		return err
	}

	if err := op.CreateOrUpdateExports(ctx, exportedValues); err != nil {
		a.Log().Error(err, "error during export validation")
		return err
	}

	// update import status
	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

	// as all exports are validated, lets trigger dependant components
	// todo: check if this is a must, maybe track what we already successfully triggered
	// maybe we also need to increase the generation manually to signal a new config version
	if err := op.TriggerDependants(ctx); err != nil {
		a.Log().Error(err, "error during dependant trigger")
		return err
	}
	return nil
}

func (a *actuator) StartNewReconcile(ctx context.Context, op *installations.Operation, inst *installations.Installation) error {
	validator := imports.NewValidator(op, op.Context().Parent, op.Context().Siblings...)
	if err := validator.Validate(ctx, inst); err != nil {
		a.Log().Error(err, "unable to validate imports")
		return err
	}

	// as all imports are satisfied we can collect and merge all imports
	// and then start the executions

	// only needed if execution are processed
	constructor := imports.NewConstructor(op, op.Context().Parent, op.Context().Siblings...)
	importDataObjects, importedValues, err := constructor.Construct(ctx, inst)
	if err != nil {
		a.Log().Error(err, "unable to collect imports")
		return err
	}

	if err := op.CreateOrUpdateImports(ctx, importDataObjects); err != nil {
		a.Log().Error(err, "unable to update import objects")
		return err
	}

	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	inst.Info.Status.Imports = inst.ImportStatus().GetStates()
	if err := op.SetExportConfigGeneration(ctx); err != nil {
		return err
	}

	subinstallation := subinstallations.New(op)
	if err := subinstallation.Ensure(ctx, inst.Info, inst.Blueprint); err != nil {
		a.Log().Error(err, "unable to ensure sub installations")
		return err
	}

	if err := subinstallation.TriggerSubInstallations(ctx, inst.Info, lsv1alpha1.ReconcileOperation); err != nil {
		return err
	}

	exec := executions.New(op)
	if err := exec.Ensure(ctx, inst, importedValues); err != nil {
		a.Log().Error(err, "unable to ensure execution")
		return err
	}
	return nil
}
