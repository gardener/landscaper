// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

var (
	SiblingImportError = errors.New("a sibling still imports some of the exports") //nolint:staticcheck
	SiblingDeleteError = errors.New("deletion of a sibling failed")                //nolint:staticcheck
)

func (c *Controller) handleDeletionPhaseInit(ctx context.Context, inst *lsv1alpha1.Installation) (fatalError lserrors.LsError, normalError lserrors.LsError) {
	op := "handleDeletionPhaseInit"

	fatalError, normalError = c.deleteAllowed(ctx, inst)
	if fatalError != nil || normalError != nil {
		return fatalError, normalError
	}

	exec, err := executions.GetExecutionForInstallation(ctx, c.LsUncachedClient(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "GetExecutionForInstallation", err.Error()), nil
	}

	if exec != nil {
		if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(inst.ObjectMeta) &&
			!lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(exec.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			if err := c.WriterToLsUncachedClient().UpdateExecution(ctx, read_write_layer.W000102, exec); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "UpdateExecutionConflict", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "UpdateExecution", err.Error()), nil
			}
		}

		if exec.DeletionTimestamp.IsZero() {
			if err = c.WriterToLsUncachedClient().DeleteExecution(ctx, read_write_layer.W000012, exec); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "DeleteExecutionConflict", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "DeleteExecution", err.Error()), nil
			}
		}
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.LsUncachedClient(), inst, inst.Status.SubInstCache, read_write_layer.R000085)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListSubinstallations", err.Error()), nil
	}

	for _, subInst := range subInsts {
		if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(inst.ObjectMeta) &&
			!lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(subInst.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			if err := c.WriterToLsUncachedClient().UpdateInstallation(ctx, read_write_layer.W000103, subInst); err != nil {
				if apierrors.IsConflict(err) {
					return nil, lserrors.NewWrappedError(err, op, "UpdateInstallationConflict", err.Error())
				}
				return lserrors.NewWrappedError(err, op, "UpdateInstallation", err.Error()), nil
			}
		}

		if subInst.DeletionTimestamp.IsZero() {
			if err = c.WriterToLsUncachedClient().DeleteInstallation(ctx, read_write_layer.W000091, subInst); err != nil {
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
	exec, err := executions.GetExecutionForInstallation(ctx, c.LsUncachedClient(), inst)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "GetExecutionForInstallation", err.Error())
	}

	if exec != nil && exec.Status.JobID != inst.Status.JobID {
		exec.Status.JobID = inst.Status.JobID
		exec.Status.TransitionTimes = lsutil.NewTransitionTimes()
		if err = c.WriterToLsUncachedClient().UpdateExecutionStatus(ctx, read_write_layer.W000093, exec); err != nil {
			return lserrors.NewWrappedError(err, op, "UpdateExecutionStatus", err.Error())
		}
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.LsUncachedClient(), inst, inst.Status.SubInstCache, read_write_layer.R000088)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListSubinstallations", err.Error())
	}

	for _, subInst := range subInsts {
		if subInst.Status.JobID != inst.Status.JobID {
			subInst.Status.JobID = inst.Status.JobID
			subInst.Status.TransitionTimes = lsutil.NewTransitionTimes()
			if err = c.WriterToLsUncachedClient().UpdateInstallationStatus(ctx, read_write_layer.W000094, subInst); err != nil {
				return lserrors.NewWrappedError(err, op, "UpdateInstallationStatus", err.Error())
			}
		}
	}

	return nil
}

func (c *Controller) handleDeletionPhaseDeleting(ctx context.Context, inst *lsv1alpha1.Installation) (allFinished bool, allDeleted bool, lsErr lserrors.LsError) {
	op := "handleDeletionPhaseDeleting"
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	exec, err := executions.GetExecutionForInstallation(ctx, c.LsUncachedClient(), inst)
	if err != nil {
		return false, false, lserrors.NewWrappedError(err, op, "GetExecutionForInstallation", err.Error())
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.LsUncachedClient(), inst, inst.Status.SubInstCache, read_write_layer.R000091)
	if err != nil {
		return false, false, lserrors.NewWrappedError(err, op, "ListSubinstallations", err.Error())
	}

	if exec == nil && len(subInsts) == 0 {
		controllerutil.RemoveFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err = c.WriterToLsUncachedClient().UpdateInstallation(ctx, read_write_layer.W000095, inst); err != nil {
			return false, false, lserrors.NewWrappedError(err, op, "UpdateInstallation", err.Error())
		}

		if inst.Spec.Optimization == nil || !inst.Spec.Optimization.HasNoSiblingImports {
			// touch siblings to speed up processing
			// a potential improvement is to only touch siblings exporting data for the current installation but this would
			// result in more complex coding and should only be done if the current approach results in performance problems
			_, siblings, err := installations.GetParentAndSiblings(ctx, c.LsUncachedClient(), inst)
			if err != nil {
				return false, false, lserrors.NewWrappedError(err, op, "GetParentAndSiblings", err.Error())
			}
			for _, nextSibling := range siblings {
				if !nextSibling.DeletionTimestamp.IsZero() {
					lsv1alpha1helper.Touch(&nextSibling.ObjectMeta)
					if err = c.WriterToLsUncachedClient().UpdateInstallation(ctx, read_write_layer.W000147, nextSibling); err != nil {
						if apierrors.IsConflict(err) {
							logger.Info(op + " - conflict touching sibling inst")
						} else if apierrors.IsNotFound(err) {
							logger.Info(op + " - not found touching sibling inst")
						} else {
							return false, false, lserrors.NewWrappedError(err, op, "TouchSibling", err.Error())
						}
					}
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

func (c *Controller) deleteAllowed(ctx context.Context, inst *lsv1alpha1.Installation) (fatalError lserrors.LsError, normalError lserrors.LsError) {
	op := "DeleteInstallationAllowed"

	v, ok := inst.GetAnnotations()[lsv1alpha1.DeleteIgnoreSuccessors]
	if ok && v == "true" {
		return nil, nil
	}

	if inst.Spec.Optimization != nil && inst.Spec.Optimization.HasNoSiblingExports {
		return nil, nil
	}

	_, siblings, err := installations.GetParentAndSiblings(ctx, c.LsUncachedClient(), inst)
	if err != nil {
		return nil, lserrors.NewWrappedError(err,
			op, "CalculateInstallationContext", err.Error(), lsv1alpha1.ErrorInternalProblem)
	}

	// check if suitable for deletion
	return checkIfSiblingImports(inst, installations.CreateInternalInstallationBases(siblings...))
}

// checkIfSiblingImports checks if a sibling imports any of the installations exports.
func checkIfSiblingImports(inst *lsv1alpha1.Installation, siblings []*installations.InstallationAndImports) (fatalError lserrors.LsError, normalError lserrors.LsError) {
	for _, sibling := range siblings {
		if inst.IsSuccessor(sibling.GetInstallation()) {
			return checkSuccessorSibling(inst, sibling)
		}
	}

	return nil, nil
}

// checkSuccessorSibling is called during the deletion of an installation (parameter "inst")
// for each successor sibling that still exists (parameter "sibling"). There are two cases:
//   - If the deletion of "sibling" has failed (with same jobID), the deletion of "inst" must also fail.
//     This is achieved by a fatal error.
//   - Otherwise, the existence of "sibling" means that "inst" cannot yet be deleted, but must be checked again later.
//     This is achieved by a normal error.
func checkSuccessorSibling(inst *lsv1alpha1.Installation,
	sibling *installations.InstallationAndImports) (fatalError lserrors.LsError, normalError lserrors.LsError) {

	op := "CheckSuccessorSibling"

	if inst.Status.JobID == sibling.GetInstallation().Status.JobIDFinished &&
		sibling.GetInstallation().Status.InstallationPhase == lsv1alpha1.InstallationPhases.DeleteFailed {

		err := lserrors.NewWrappedError(SiblingDeleteError, op, "SiblingDeleteError",
			SiblingDeleteError.Error(), lsv1alpha1.ErrorForInfoOnly)

		return err, nil
	}

	err := lserrors.NewWrappedError(SiblingImportError, op, "SiblingImport", SiblingImportError.Error(),
		lsv1alpha1.ErrorForInfoOnly)

	return nil, err
}
