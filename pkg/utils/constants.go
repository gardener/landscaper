package utils

import lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

func IsDeployItemPhase(di *lsv1alpha1.DeployItem, phase lsv1alpha1.DeployItemPhase) bool {
	return di.Status.DeployItemPhase == phase
}

func IsInstallationPhase(inst *lsv1alpha1.Installation, phase lsv1alpha1.InstallationPhase) bool {
	return inst.Status.InstallationPhase == phase
}

func IsDeployItemJobIDsIdentical(di *lsv1alpha1.DeployItem) bool {
	return di.Status.GetJobID() == di.Status.JobIDFinished
}

func IsInstallationJobIDsIdentical(inst *lsv1alpha1.Installation) bool {
	return inst.Status.JobID == inst.Status.JobIDFinished
}

func IsExecutionJobIDsIdentical(exec *lsv1alpha1.Execution) bool {
	return exec.Status.JobID == exec.Status.JobIDFinished
}
