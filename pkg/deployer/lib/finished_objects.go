package lib

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

func IsDeployItemFinished(di *lsv1alpha1.DeployItem) bool {
	hasTestReconcileAnnotation := lsv1alpha1helper.HasOperation(di.ObjectMeta, lsv1alpha1.TestReconcileOperation)
	return !hasTestReconcileAnnotation && di.Status.GetJobID() == di.Status.JobIDFinished
}
