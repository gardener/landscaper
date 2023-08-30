package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// NewTransitionTimes creates a TransitionTimes structure. The status of Installations, Executions, and DeployItems
// contain such a structure with important timestamps. The timestamps are intended to help in the analysis of errors.
//   - The TriggerTime is set when the object is triggered, i.e. when it gets a new jobID.
//   - The InitTime is set when the responsible controller observes that an object has a new jobID, starts processing it,
//     and sets its phase to Init.
//   - The WaitTime is set when the controller has triggered its subobjects (resp. has done its deployment) and starts
//     waiting for its subobjects (resp. for the readiness check).
//   - The FinishedTime is set when an object gets into a final phase (Succeeded, Failed, DeleteFailed) and the
//     jobIDFinished is set.
func NewTransitionTimes() *lsv1alpha1.TransitionTimes {
	now := metav1.Now()
	return &lsv1alpha1.TransitionTimes{
		TriggerTime: &now,
	}
}

func SetInitTransitionTime(transitionTimes *lsv1alpha1.TransitionTimes) *lsv1alpha1.TransitionTimes {
	if transitionTimes == nil {
		transitionTimes = &lsv1alpha1.TransitionTimes{}
	}

	now := metav1.Now()
	transitionTimes.InitTime = &now
	return transitionTimes
}

func SetWaitTransitionTime(transitionTimes *lsv1alpha1.TransitionTimes) *lsv1alpha1.TransitionTimes {
	if transitionTimes == nil {
		transitionTimes = &lsv1alpha1.TransitionTimes{}
	}

	now := metav1.Now()
	transitionTimes.WaitTime = &now
	return transitionTimes
}

func SetFinishedTransitionTime(transitionTimes *lsv1alpha1.TransitionTimes) *lsv1alpha1.TransitionTimes {
	if transitionTimes == nil {
		transitionTimes = &lsv1alpha1.TransitionTimes{}
	}

	now := metav1.Now()
	transitionTimes.FinishedTime = &now
	return transitionTimes
}
