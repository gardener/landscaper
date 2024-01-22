package execution

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

func isExecFinished(exec *lsv1alpha1.Execution) bool {
	if needsFinalizer(exec) ||
		hasInterruptOperation(exec) ||
		isDifferentJobIDs(exec) {
		return false
	}

	return true
}

func needsFinalizer(exec *lsv1alpha1.Execution) bool {
	return exec.DeletionTimestamp.IsZero() && !kutil.HasFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
}

func hasInterruptOperation(exec *lsv1alpha1.Execution) bool {
	return lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.InterruptOperation)
}

func isDifferentJobIDs(exec *lsv1alpha1.Execution) bool {
	return exec.Status.JobID != exec.Status.JobIDFinished
}
