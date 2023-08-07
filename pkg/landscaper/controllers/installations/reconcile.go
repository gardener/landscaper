// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	lspatch "github.com/gardener/landscaper/pkg/utils/patch"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (c *Controller) handleReconcilePhase(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	op := "handleReconcilePhase"

	// set init phase if the phase is empty or final from previous job
	if inst.Status.InstallationPhase.IsFinal() || inst.Status.InstallationPhase.IsEmpty() {

		nextPhase := lsv1alpha1.InstallationPhases.Init
		if !inst.DeletionTimestamp.IsZero() {
			nextPhase = lsv1alpha1.InstallationPhases.InitDelete
		}

		inst.Status.InstallationPhase = nextPhase
		now := metav1.Now()
		inst.Status.PhaseTransitionTime = &now
		inst.Status.TransitionTimes = lsutil.SetInitTransitionTime(inst.Status.TransitionTimes)
		inst.Status.ObservedGeneration = inst.GetGeneration()

		// do not use setInstallationPhaseAndUpdate because jobIDFinished should not be set here
		if err := c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000115, inst); err != nil {
			return lserrors.NewWrappedError(err, op, "InitialPhaseSetting", err.Error())
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Init {
		fatalError, normalError := c.handlePhaseInit(ctx, inst)

		inst.Status.ObservedGeneration = inst.GetGeneration()

		if fatalError != nil && !lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.Failed, fatalError, read_write_layer.W000087)
		} else if fatalError != nil && lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, fatalError, read_write_layer.W000003)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000088)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.CleanupOrphaned, nil, read_write_layer.W000114); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.CleanupOrphaned {
		fatalError, normalError := c.handlePhaseCleanupOrphaned(ctx, inst)

		if fatalError != nil && !lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.Failed, fatalError, read_write_layer.W000019)
		} else if fatalError != nil && lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, fatalError, read_write_layer.W000023)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000024)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.ObjectsCreated, nil, read_write_layer.W000025); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.ObjectsCreated {
		if err := c.handlePhaseObjectsCreated(ctx, inst); err != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000116)
		}

		inst.Status.TransitionTimes = lsutil.SetWaitTransitionTime(inst.Status.TransitionTimes)

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.Progressing, nil, read_write_layer.W000117); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Progressing {
		allSucceeded, err := c.handlePhaseProgressing(ctx, inst)
		if err != nil {
			// error or unfinished subobjects => phase remains progressing
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000118)
		}

		var nextPhase lsv1alpha1.InstallationPhase
		if allSucceeded {
			nextPhase = lsv1alpha1.InstallationPhases.Completing
		} else {
			nextPhase = lsv1alpha1.InstallationPhases.Failed
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, nextPhase, nil, read_write_layer.W000119); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Completing {
		fatalError, normalError := c.handlePhaseCompleting(ctx, inst)

		if fatalError != nil && !lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.Failed, fatalError, read_write_layer.W000120)
		} else if fatalError != nil && lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, fatalError, read_write_layer.W000005)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000121)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.Succeeded, nil, read_write_layer.W000122); err != nil {
			return err
		}

		return nil
	}

	// handle deletion phases

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.InitDelete {
		// trigger deletion of execution and sub installations
		fatalError, normalError := c.handleDeletionPhaseInit(ctx, inst)

		if fatalError != nil && !lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.DeleteFailed, fatalError, read_write_layer.W000123)
		} else if fatalError != nil && lsutil.IsRecoverableError(fatalError) {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, fatalError, read_write_layer.W000006)
		} else if normalError != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, normalError, read_write_layer.W000124)
		}

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.TriggerDelete, nil, read_write_layer.W000125); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.TriggerDelete {

		if err := c.handleDeletionPhaseTriggerDeleting(ctx, inst); err != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000126)
		}

		inst.Status.TransitionTimes = lsutil.SetWaitTransitionTime(inst.Status.TransitionTimes)

		if err := c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.Deleting, nil, read_write_layer.W000127); err != nil {
			return err
		}
	}

	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Deleting {
		// wait until all sub objects are gone or finished

		allFinished, allDeleted, err := c.handleDeletionPhaseDeleting(ctx, inst)

		if err != nil {
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000128)
		} else if allDeleted {
			return nil
		} else if allFinished {
			err = lserrors.NewError(op, "UndeletedSubobjects", "not all sub objects were deleted", lsv1alpha1.ErrorForInfoOnly)
			return c.setInstallationPhaseAndUpdate(ctx, inst, lsv1alpha1.InstallationPhases.DeleteFailed, err, read_write_layer.W000129)
		} else {
			// retry
			err = lserrors.NewError(op, "PendingSubobjects", "deletion of some sub objects pending", lsv1alpha1.ErrorForInfoOnly)
			return c.setInstallationPhaseAndUpdate(ctx, inst, inst.Status.InstallationPhase, err, read_write_layer.W000130)
		}
	}

	return nil
}

func (c *Controller) handlePhaseInit(ctx context.Context, inst *lsv1alpha1.Installation) (lserrors.LsError, lserrors.LsError) {
	currentOperation := "handlePhaseInit"

	// cleanup
	newCleaner := NewDataObjectAndTargetCleaner(inst, c.Client())
	if err := newCleaner.CleanupContext(ctx); err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "CleanupContext", err.Error()), nil
	}
	if err := newCleaner.CleanupExports(ctx); err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "CleanupExports", err.Error()), nil
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
	*imports.Imports, string, map[string]*installations.InstallationAndImports, lserrors.LsError, lserrors.LsError) {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

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

	predecessors := rh.FetchPredecessors()

	predecessorMap, err := rh.GetPredecessors(inst, predecessors)
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

	imps, err := rh.ImportsSatisfied()
	if err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "ImportsSatisfied", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	hash, err := c.hash(imps)
	if err != nil {
		fatalError = lserrors.NewWrappedError(err, currentOperation, "HashImports", err.Error())
		return nil, nil, "", nil, fatalError, nil
	}

	logger.Debug("imports hash computation", "hash", hash)

	return instOp, imps, hash, predecessorMap, nil, nil
}

func (c *Controller) hash(imps *imports.Imports) (string, error) {
	hash, err := imports.ComputeImportsHash(imps)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func (c *Controller) handlePhaseCleanupOrphaned(ctx context.Context, inst *lsv1alpha1.Installation) (lserrors.LsError, lserrors.LsError) {
	currentOperation := "handlePhaseCleanupOrphaned"

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, currentOperation, "ListSubinstallations", err.Error())
	}

	subInstsToDelete := []*lsv1alpha1.Installation{}
	for _, next := range subInsts {
		if !next.ObjectMeta.DeletionTimestamp.IsZero() {
			subInstsToDelete = append(subInstsToDelete, next)
		}
	}

	if len(subInstsToDelete) == 0 {
		return nil, nil
	}

	// Consider the remaining subinstallations to be deleted. If they are all finished with phase DeleteFailed,
	// we are stuck and return an error.
	allFailed := true
	for _, next := range subInstsToDelete {
		if next.Status.JobIDFinished != inst.Status.JobID || next.Status.InstallationPhase != lsv1alpha1.InstallationPhases.DeleteFailed {
			allFailed = false
		}
	}
	if allFailed {
		return lserrors.NewError(currentOperation, "AllOrphanedSubinstallationsFailed", "all orphaned subinstallations failed"), nil
	}

	for _, next := range subInstsToDelete {
		if next.Status.JobID != inst.Status.JobID {
			next.Status.JobID = inst.Status.JobID
			inst.Status.TransitionTimes = lsutil.NewTransitionTimes()
			if err = c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000076, next); err != nil {
				return nil, lserrors.NewWrappedError(err, currentOperation, "UpdateInstallationStatus", err.Error())
			}
		}
	}

	return nil, lserrors.NewError(currentOperation, "OrphanedSubinstsStillDeleting",
		"some orphaned subinstallations are still deleting")
}

func (c *Controller) handlePhaseObjectsCreated(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	currentOperation := "handlePhaseObjectsCreated"

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ListSubinstallations", err.Error())
	}

	// cleanup references in status
	oldReferences := inst.Status.InstallationReferences
	newReferences := []lsv1alpha1.NamedObjectReference{}
	for _, nextRef := range oldReferences {
		for _, nextSubInst := range subInsts {
			if nextSubInst.Name == nextRef.Reference.Name {
				newReferences = append(newReferences, nextRef)
				break
			}
		}
	}

	inst.Status.InstallationReferences = newReferences
	if err := c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000026, inst); err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "UpdateInstallationReferences", err.Error())
	}

	// trigger subinstallations
	for _, next := range subInsts {
		if next.Status.JobID != inst.Status.JobID {
			patch := lspatch.NewPatch(map[string]any{
				"metadata": map[string]any{
					"resourceVersion": next.GetResourceVersion(), // optimistic locking
				},
				"status": map[string]any{
					"jobID":              inst.Status.JobID,
					"transitionTimes":    lsutil.NewTransitionTimes(),
					"configGeneration":   next.Status.ConfigGeneration,   // ensure required field in new status
					"observedGeneration": next.Status.ObservedGeneration, // ensure required field in new status
				},
			})
			if err = c.Writer().PatchInstallationStatus(ctx, read_write_layer.W000083, next, patch); err != nil {
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
			patch := lspatch.NewPatch(map[string]any{
				"metadata": map[string]any{
					"resourceVersion": exec.GetResourceVersion(), // optimistic locking
				},
				"status": map[string]any{
					"jobID":              inst.Status.JobID,
					"transitionTimes":    lsutil.NewTransitionTimes(),
					"observedGeneration": exec.Status.ObservedGeneration, // ensure required field in new status
				},
			})
			if err := c.Writer().PatchExecutionStatus(ctx, read_write_layer.W000084, exec, patch); err != nil {
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
			return false, lserrors.NewError(currentOperation, "JobIDFinished", message,
				lsv1alpha1.ErrorUnfinished, lsv1alpha1.ErrorForInfoOnly)
		}

		allSucceeded = allSucceeded && (next.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Succeeded)
	}

	if inst.Status.ExecutionReference != nil {
		key := client.ObjectKey{Namespace: inst.Status.ExecutionReference.Namespace, Name: inst.Status.ExecutionReference.Name}
		exec := &lsv1alpha1.Execution{}
		if err := read_write_layer.GetExecution(ctx, c.Client(), key, exec); err != nil {
			return false, lserrors.NewWrappedError(err, currentOperation, "GetExecution", err.Error())
		}

		if exec.Status.JobIDFinished != exec.Status.JobID {
			message := fmt.Sprintf("execution %s / %s is not finished yet", exec.Namespace, exec.Name)
			return false, lserrors.NewError(currentOperation, "JobIDFinished", message,
				lsv1alpha1.ErrorUnfinished, lsv1alpha1.ErrorForInfoOnly)
		}

		allSucceeded = allSucceeded && (exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.Succeeded)
	}

	return allSucceeded, nil
}

func (c *Controller) handlePhaseCompleting(ctx context.Context, inst *lsv1alpha1.Installation) (lserrors.LsError, lserrors.LsError) {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

	currentOperation := "handlePhaseCompleting"

	instOp, imps, importsHash, _, fatalError, fatalError2 := c.init(ctx, inst)

	if fatalError != nil {
		return fatalError, nil
	} else if fatalError2 != nil {
		return fatalError2, nil
	}

	if importsHash != inst.Status.ImportsHash {
		logger.Info("changed hash", "oldHash", inst.Status.ImportsHash, "newHash", importsHash)
		return lserrors.NewError(currentOperation, "CheckImportsHash", "imports have changed"), nil
	}

	if inst.Generation != inst.Status.ObservedGeneration {
		return lserrors.NewError(currentOperation, "CheckObservedGeneration", "installation spec has been changed", lsv1alpha1.ErrorForInfoOnly), nil
	}

	con := imports.NewConstructor(instOp)
	err := con.Construct(ctx, imps)
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "ConstructImportsForExports", err.Error()), nil
	}
	err = con.RenderImportExecutions()
	if err != nil {
		return lserrors.NewWrappedError(err, currentOperation, "RenderImportExecutionsForExports", err.Error()), nil
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

	return nil, nil
}

func (c *Controller) CreateImportsAndSubobjects(ctx context.Context, op *installations.Operation, imps *imports.Imports) lserrors.LsError {
	inst := op.Inst
	currOp := "CreateImportsAndSubobjects"
	// collect and merge all imports and start the Executions
	constructor := imports.NewConstructor(op)
	if err := constructor.Construct(ctx, imps); err != nil {
		return lserrors.NewWrappedError(err, currOp, "ConstructImports", err.Error())
	}
	if err := constructor.RenderImportExecutions(); err != nil {
		return lserrors.NewWrappedError(err, currOp, "RenderImportExecutions", err.Error())
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

	inst.GetInstallation().Status.Imports = inst.ImportStatus().GetStatus()
	return nil
}

func (c *Controller) removeReconcileAnnotation(ctx context.Context, inst *lsv1alpha1.Installation) lserrors.LsError {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		logger.Debug("remove reconcile annotation")
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		delete(inst.Annotations, lsv1alpha1.ReconcileReasonAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000009, inst); client.IgnoreNotFound(err) != nil {
			return lserrors.NewWrappedError(err, "RemoveReconcileAnnotation", "UpdateInstallation", err.Error())
		}
	}
	return nil
}
