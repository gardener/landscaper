package utils

import lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

var NewReconcile = true

func IsNewReconcile() bool {
	return NewReconcile
}

func IsDeployItemPhase(di *lsv1alpha1.DeployItem, phase lsv1alpha1.DeployItemPhase) bool {
	return !IsNewReconcile() || di.Status.DeployItemPhase == phase
}

func IsInstallationPhase(inst *lsv1alpha1.Installation, phase lsv1alpha1.InstallationPhase) bool {
	return !IsNewReconcile() || inst.Status.InstallationPhase == phase
}

func IsExecutionPhase(exec *lsv1alpha1.Execution, phase lsv1alpha1.ExecPhase) bool {
	return !IsNewReconcile() || exec.Status.ExecutionPhase == phase
}

func IsDeployItemJobIDsIdentical(di *lsv1alpha1.DeployItem) bool {
	return !IsNewReconcile() || di.Status.JobID == di.Status.JobIDFinished
}

func IsInstallationJobIDsIdentical(inst *lsv1alpha1.Installation) bool {
	return !IsNewReconcile() || inst.Status.JobID == inst.Status.JobIDFinished
}

func IsExecutionJobIDsIdentical(exec *lsv1alpha1.Execution) bool {
	return !IsNewReconcile() || exec.Status.JobID == exec.Status.JobIDFinished
}
