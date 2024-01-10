package installations

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

func isInstFinished(inst *lsv1alpha1.Installation) bool {
	if isAutomaticReconcileOnSpecChange(inst) ||
		isAutomaticReconcileConfigured(inst) ||
		needsFinalizer(inst) ||
		hasDependentsToTrigger(inst) ||
		hasInterruptOperation(inst) ||
		isNotRootWithReconcileOperation(inst) ||
		isCreateNewJobID(inst) ||
		isDifferentJobIDs(inst) {
		return false
	}

	return true
}

func isAutomaticReconcileOnSpecChange(inst *lsv1alpha1.Installation) bool {
	return installations.IsRootInstallation(inst) &&
		lsv1alpha1helper.HasReconcileIfChangedAnnotation(inst.ObjectMeta) &&
		!lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) &&
		inst.Status.JobID == inst.Status.JobIDFinished &&
		inst.GetGeneration() != inst.Status.ObservedGeneration
}

func isAutomaticReconcileConfigured(inst *lsv1alpha1.Installation) bool {
	retryHelper := newRetryHelper(nil, nil)
	return retryHelper.isRetryActivatedForSucceeded(inst) || retryHelper.isRetryActivatedForFailed(inst)
}

func needsFinalizer(inst *lsv1alpha1.Installation) bool {
	return inst.DeletionTimestamp.IsZero() && !kutil.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer)

}

func hasDependentsToTrigger(inst *lsv1alpha1.Installation) bool {
	return len(inst.Status.DependentsToTrigger) > 0
}

func hasInterruptOperation(inst *lsv1alpha1.Installation) bool {
	return lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.InterruptOperation)
}

func isNotRootWithReconcileOperation(inst *lsv1alpha1.Installation) bool {
	return !installations.IsRootInstallation(inst) && lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
}

func isCreateNewJobID(inst *lsv1alpha1.Installation) bool {
	isFirstDelete := !inst.DeletionTimestamp.IsZero() && !inst.Status.InstallationPhase.IsDeletion()

	return installations.IsRootInstallation(inst) &&
		(lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) || isFirstDelete) &&
		inst.Status.JobID == inst.Status.JobIDFinished
}

func isDifferentJobIDs(inst *lsv1alpha1.Installation) bool {
	return inst.Status.JobID != inst.Status.JobIDFinished
}
