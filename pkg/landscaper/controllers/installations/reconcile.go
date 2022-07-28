// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (c *Controller) handleReconcilePhase(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	op := "handleReconcilePhase"

	// set init phase if the phase is empty or final from previous job
	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseSucceeded ||
		inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseFailed ||
		inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseDeleteFailed ||
		inst.Status.InstallationPhase == "" {

		nextPhase := lsv1alpha1.InstallationPhaseInit
		if !inst.DeletionTimestamp.IsZero() {
			nextPhase = lsv1alpha1.InstallationPhaseInitDelete
		}

		inst.Status.InstallationPhase = nextPhase

		// do not use setInstallationPhaseAndUpdate because jobIDFinished should not be set here
		if err := c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000115, inst); err != nil {
			return lserrors.NewWrappedError(err, op, "InitialPhaseSetting", err.Error())
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseInit {
		fatalError, normalError := c.handlePhaseInit(ctx, inst)

		if fatalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseFailed, fatalError, read_write_layer.W000087)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000088)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseObjectsCreated, nil, read_write_layer.W000114); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseObjectsCreated {
		if err := c.handlePhaseObjectsCreated(ctx, inst); err != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000116)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseProgressing, nil, read_write_layer.W000117); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseProgressing {
		allSucceeded, err := c.handlePhaseProgressing(ctx, inst)
		if err != nil {
			// error or unfinished subobjects => phase remains progressing
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000118)
		}

		var nextPhase lsv1alpha1.InstallationPhase
		if allSucceeded {
			nextPhase = lsv1alpha1.InstallationPhaseCompleting
		} else {
			nextPhase = lsv1alpha1.InstallationPhaseFailed
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, nextPhase, nil, read_write_layer.W000119); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseCompleting {
		fatalError, normalError := c.handlePhaseCompleting(ctx, inst)

		if fatalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseFailed, fatalError, read_write_layer.W000120)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000121)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseSucceeded, nil, read_write_layer.W000122); err != nil {
			return err
		}
	}

	// handle deletion phases

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseInitDelete {
		// trigger deletion of execution and sub installations
		fatalError, normalError := c.handleDeletionPhaseInit(ctx, inst)

		if fatalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseDeleteFailed, fatalError, read_write_layer.W000123)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000124)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseTriggerDelete, nil, read_write_layer.W000125); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseTriggerDelete {

		if err := c.handleDeletionPhaseTriggerDeleting(ctx, inst); err != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000126)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseDeleting, nil, read_write_layer.W000127); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseDeleting {
		// wait until all sub objects are gone or finished

		allFinished, allDeleted, err := c.handleDeletionPhaseDeleting(ctx, inst)

		if err != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000128)
		} else if allDeleted {
			return nil
		} else if allFinished {
			err = lserrors.NewError(op, "UndeletedSubobjects", "not all sub objects were deleted")
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhaseDeleteFailed, err, read_write_layer.W000129)
		} else {
			// retry
			err = lserrors.NewError(op, "PendingSubobjects", "deletion of some sub objects pending")
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000130)
		}
	}

	return nil
}

func (c *Controller) handlePhaseInit(ctx context.Context, inst *lsv1alpha1.Installation) (lserrors.LsError, lserrors.LsError) {
	currentOperation := "handlePhaseInit"

	err := c.checkForDuplicateExports(ctx, inst)
	if err != nil {
		return lserrors.BuildLsError(err, currentOperation, "CheckForDuplicateExports", err.Error(), lsv1alpha1.ErrorConfigurationProblem), nil
	}

	instOp, imps, importsHash, predecessorMap, fatalError, normalError := c.init(ctx, inst)

	if fatalError != nil {
		return fatalError, nil
	} else if normalError != nil {
		return nil, normalError
	}

	if err := c.CreateImportsAndSubobjects(ctx, instOp, imps); err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "CreateImportsAndSubobjects", err.Error()), nil
	}

	// we need to recheck the predecessors because they might have been changed during fetching the import data and therefore
	// the import data might not be consistent. Then we should not go to the next phase and start the current sub objects
	// fatal errors are not so important here as there will be a retry and if these still exists, they will result in a failure
	// in the next reconcile loop
	_, _, importsHashNew, predecessorMapNew, fatalError, normalError := c.init(ctx, inst)
	if fatalError != nil {
		return nil, fatalError
	} else if normalError != nil {
		return nil, normalError
	}

	allSame := c.compareJobIDs(predecessorMap, predecessorMapNew)
	if !allSame {
		return nil, lserrors.NewError(currentOperation, "comparePredecessorMaps", "some predecessor was changed during fetching the import data")
	}

	if importsHashNew != importsHash {
		return nil, lserrors.NewError(currentOperation, "compareImportHashes", "some predecessor was changed during fetching the import data")
	}

	inst.Status.ImportsHash = importsHash

	return nil, nil
}

func (c *Controller) init(ctx context.Context, inst *lsv1alpha1.Installation) (*installations.Operation,
	*imports.Imports, string, map[string]*installations.InstallationBase, lserrors.LsError, lserrors.LsError) {
	currentOperation := "init"

	instOp, fatalError := c.initPrerequisites(ctx, inst)
	if fatalError != nil {
		return nil, nil, "", nil, fatalError, nil
	}

	instOp.CurrentOperation = currentOperation

	rh, err := reconcilehelper.NewReconcileHelper(ctx, instOp)
	if err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "NewReconcileHelper", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	dependendOnSiblings, err := rh.FetchDependencies()
	if err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "FetchDependencies", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	predecessorMap, err := rh.GetPredecessors(inst, dependendOnSiblings)
	if err != nil {
		normalError := lserrors.NewWrappedError(err, currentOperation, "GetPredecessors", err.Error())
		return nil, nil, "", nil, nil, normalError
	}

	if err = rh.AllPredecessorsFinished(inst, predecessorMap); err != nil {
		normalError := lserrors.NewWrappedError(err, currentOperation, "AllPredecessorsFinished", err.Error())
		return nil, nil, "", nil, nil, normalError
	}

	if err = rh.AllPredecessorsSucceeded(inst, predecessorMap); err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "AllPredecessorsSucceeded", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	if err = rh.ImportsSatisfied(); err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "ImportsSatisfied", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	imps, err := rh.GetImports()
	if err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "GetImports", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	hash, err := c.hash(imps)
	if err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "HashImports", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	c.Log().Debug("imports hash computation", "hash", hash)

	return instOp, imps, hash, predecessorMap, nil, nil
}

func (c *Controller) hash(imps *imports.Imports) (string, error) {
	hash, err := imports.ComputeImportsHash(imps)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func (c *Controller) handlePhaseObjectsCreated(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	currentOperation := "handlePhaseObjectsCreated"

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ListSubinstallations", err.Error())
	}

	// trigger subinstallations
	for _, next := range subInsts {
		if next.Status.JobID != inst.Status.JobID {
			next.Status.JobID = inst.Status.JobID
			if err = c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000083, next); err != nil {
				return lserrors.NewWrappedError(err, currentOperation, "UpdateInstallationStatus", err.Error())
			}
		}
	}

	if inst.Status.ExecutionReference != nil {
		key := client.ObjectKey{Namespace: inst.Status.ExecutionReference.Namespace, Name: inst.Status.ExecutionReference.Name}
		exec := &lsv1alpha1.Execution{}
		if err := read_write_layer.GetExecution(ctx, c.Client(), key, exec); err != nil {
			return lserrors.NewWrappedError(err, currentOperation, "GetExecution", err.Error())
		}

		if exec.Status.JobID != inst.Status.JobID {
			exec.Status.JobID = inst.Status.JobID
			if err := c.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000084, exec); err != nil {
				return lserrors.NewWrappedError(err, currentOperation, "UpdateExecutionStatus", err.Error())
			}
		}
	}

	return nil
}

func (c *Controller) handlePhaseProgressing(ctx context.Context, inst *lsv1alpha1.Installation) (allSucceeded bool, lsErr lserrors.LsError) {
	currentOperation := "handlePhaseProgressing"

	allSucceeded = true

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return false, lserrors.NewWrappedError(err, currentOperation, "ListSubinstallations", err.Error())
	}

	for _, next := range subInsts {
		if next.Status.JobIDFinished != next.Status.JobID {
			// Hack: being unfinished should not be treated as an error
			message := fmt.Sprintf("installation %s / %s is not finished yet", next.Namespace, next.Name)
			return false, lserrors.NewError(currentOperation, "JobIDFinished", message)
		}

		allSucceeded = allSucceeded && (next.Status.InstallationPhase == lsv1alpha1.InstallationPhaseSucceeded)
	}

	if inst.Status.ExecutionReference != nil {
		key := client.ObjectKey{Namespace: inst.Status.ExecutionReference.Namespace, Name: inst.Status.ExecutionReference.Name}
		exec := &lsv1alpha1.Execution{}
		if err := read_write_layer.GetExecution(ctx, c.Client(), key, exec); err != nil {
			return false, lserrors.NewWrappedError(err, currentOperation, "GetExecution", err.Error())
		}

		if exec.Status.JobIDFinished != exec.Status.JobID {
			message := fmt.Sprintf("execution %s / %s is not finished yet", exec.Namespace, exec.Name)
			return false, lserrors.NewError(currentOperation, "JobIDFinished", message)
		}

		allSucceeded = allSucceeded && (exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseSucceeded)
	}

	return allSucceeded, nil
}

func (c *Controller) handlePhaseCompleting(ctx context.Context, inst *lsv1alpha1.Installation) (lserrors.LsError, lserrors.LsError) {
	currentOperation := "handlePhaseCompleting"

	instOp, imps, importsHash, _, fatalError, fatalError2 := c.init(ctx, inst)

	if fatalError != nil {
		return fatalError, nil
	} else if fatalError2 != nil {
		return fatalError2, nil
	}

	if importsHash != inst.Status.ImportsHash {
		c.Log().WithValues("oldHash", inst.Status.ImportsHash, "newHash", importsHash).Info("changed hash")

		return lserrors.NewError(currentOperation, "CheckImportsHash", "imports have changed"), nil
	}

	if inst.Generation != inst.Status.ObservedGeneration {
		return lserrors.NewError(currentOperation, "CheckObservedGeneration", "installation spec has been changed"), nil
	}

	err := imports.NewConstructor(instOp).Construct(ctx, imps)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ConstructImportsForExports", err.Error()), nil
	}

	dataExports, targetExports, err := exports.NewConstructor(instOp).Construct(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ConstructExports", err.Error()), nil
	}

	if err := instOp.CreateOrUpdateExports(ctx, dataExports, targetExports); err != nil {
		if apierrors.IsConflict(err) {
			return nil, lserrors.NewWrappedError(err, currentOperation, "CreateOrUpdateExports", err.Error())
		}
		return lserrors.NewWrappedError(err, currentOperation, "CreateOrUpdateExports", err.Error()), nil
	}

	if err := instOp.NewTriggerDependents(ctx); err != nil {
		if apierrors.IsConflict(err) {
			return nil, lserrors.NewWrappedError(err, currentOperation, "TriggerDependents", err.Error())
		}
		return lserrors.NewWrappedError(err, currentOperation, "TriggerDependents", err.Error()), nil
	}

	return nil, nil
}

func (c *Controller) reconcile(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	var (
		currentOperation = "Validate"
		log              = c.Log()
	)
	log.WithValues(lc.KeyMethod, "reconcile").Debug(lc.MsgStartMethod)

	combinedState, lsErr := c.combinedPhaseOfSubobjects(ctx, inst, currentOperation)
	if lsErr != nil {
		return lsErr
	}

	if !lsv1alpha1helper.IsCompletedInstallationPhase(combinedState) {
		log.Info("Waiting for all deploy items and nested installations to be completed")
		inst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
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
		return lsErr
	}
	instOp.CurrentOperation = currentOperation

	rh, err := reconcilehelper.NewReconcileHelper(ctx, instOp)
	if err != nil {
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
		return lserrors.NewWrappedError(err, currentOperation, "GetImportsForExports", err.Error())
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
	c.Log().WithValues(lc.KeyMethod, "forceReconcile").Debug(lc.MsgStartMethod)
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

func (c *Controller) CreateImportsAndSubobjects(ctx context.Context, op *installations.Operation, imps *imports.Imports) lserrors.LsError {
	inst := op.Inst
	currOp := "CreateImportsAndSubobjects"
	// collect and merge all imports and start the Executions
	constructor := imports.NewConstructor(op)
	if err := constructor.Construct(ctx, imps); err != nil {
		return lserrors.NewWrappedError(err, currOp, "ConstructImports", err.Error())
	}

	if err := op.CreateOrUpdateImports(ctx); err != nil {
		return lserrors.NewWrappedError(err, currOp, "CreateOrUpdateImports", err.Error())
	}

	subinstallation := subinstallations.New(op)
	if err := subinstallation.Ensure(ctx); err != nil {
		return lserrors.NewWrappedError(err, currOp, "EnsureSubinstallations", err.Error())
	}

	exec := executions.New(op)
	if err := exec.Ensure(ctx, inst); err != nil {
		return lserrors.NewWrappedError(err, currOp, "ReconcileExecution", err.Error())
	}

	inst.Info.Status.Imports = inst.ImportStatus().GetStatus()
	inst.Info.Status.ObservedGeneration = inst.Info.Generation
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
		c.Log().Debug("remove reconcile annotation")
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000009, inst); client.IgnoreNotFound(err) != nil {
			return lserrors.NewWrappedError(err, "RemoveReconcileAnnotation", "UpdateInstallation", err.Error())
		}
	}
	return nil
}

func (c *Controller) removeForceReconcileAnnotation(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		c.Log().Debug("remove force reconcile annotation")
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000003, inst); err != nil {
			return lserrors.NewWrappedError(err, "RemoveForceReconcileAnnotation", "UpdateInstallation", err.Error())
		}
	}
	return nil
}
