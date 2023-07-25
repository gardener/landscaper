package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

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
