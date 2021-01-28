// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StatusReady      StatusType = "Ready"
	StatusInProgress StatusType = "InProgress"
	StatusUnknown    StatusType = "Unknown"
)

// StatusType defines the value of a Status.
type StatusType string

// ObjectState represents an object and its status.
type ObjectState struct {
	Object *unstructured.Unstructured
	Status StatusType
}

// ObjectState represents an arrays of objects and their status.
type ObjectStates []*ObjectState

// GetObjectsInStatusType returns all unstructured objects in the given status.
func (o *ObjectStates) GetObjectsInStatusType(status StatusType) []*unstructured.Unstructured {
	return FilterObjectStateInStatusType(*o, status)
}

// GetObjectsNotInStatusType returns all unstructured objects not in the given status.
func (o *ObjectStates) GetObjectsNotInStatusType(status StatusType) []*unstructured.Unstructured {
	return FilterObjectStateNotInStatusType(*o, status)
}

// WaitObjectsReady waits for objects to be in ready status and
// returns an error if all the objects are not ready after the backoff duration.
func WaitObjectsReady(ctx context.Context, backoff wait.Backoff, log logr.Logger, kubeClient client.Client, objects []*unstructured.Unstructured) error {
	var (
		ready     = false
		objStates ObjectStates
	)

	currStep := 1
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		log.V(1).Info("wait resources ready", "check", fmt.Sprintf("%d/%d", currStep, backoff.Steps))
		currStep++
		var err error
		objStates, err = GetObjectStates(ctx, log, kubeClient, objects)
		if err != nil {
			return false, err
		}

		ready = len(objStates.GetObjectsInStatusType(StatusReady)) == len(objects)
		if !ready {
			return false, nil
		}
		return true, nil

	})

	if err != nil && err != wait.ErrWaitTimeout {
		return err
	}

	if !ready {
		var allErrs []error
		for _, objStates := range objStates {
			obj := objStates.Object
			status := objStates.Status
			if status != StatusReady {
				allErrs = append(allErrs, fmt.Errorf("%s %s/%s is in state %s",
					obj.GroupVersionKind().String(),
					obj.GetName(), obj.GetNamespace(),
					status))
			}
		}

		return apimacherrors.NewAggregate(allErrs)
	}

	return nil
}

// GetObjectStates gets objects from the cluster returns and ObjectStates
// with the returned objects and their status.
func GetObjectStates(ctx context.Context, log logr.Logger, kubeClient client.Client, objects []*unstructured.Unstructured) (ObjectStates, error) {
	var (
		allErrs   []error
		wg        sync.WaitGroup
		objStates = make(ObjectStates, len(objects))
	)

	for i, obj := range objects {
		wg.Add(1)
		go func(i int, obj *unstructured.Unstructured) {
			defer wg.Done()

			key := ObjectKey(obj.GetName(), obj.GetNamespace())
			if err := kubeClient.Get(ctx, key, obj); err != nil {
				allErrs = append(allErrs, fmt.Errorf("unable to get %s %s/%s: %w", obj.GroupVersionKind().String(), obj.GetName(), obj.GetNamespace(), err))
			}

			objLog := log.WithValues(
				"kind", obj.GetKind(),
				"resource", ObjectKey(obj.GetName(), obj.GetNamespace()))

			objLog.V(1).Info("get resource status")
			status, err := Status(obj)
			if err != nil {
				allErrs = append(allErrs, err)
			}
			objLog.V(1).Info("resource status", "status", status)

			objStates[i] = &ObjectState{
				Object: obj,
				Status: status,
			}

		}(i, obj)
	}
	wg.Wait()

	if len(allErrs) > 0 {
		return nil, apimacherrors.NewAggregate(allErrs)
	}

	return objStates, nil
}

// FilterObjectStateInStatusType returns the objects in the desired status.
func FilterObjectStateInStatusType(objStates ObjectStates, status StatusType) []*unstructured.Unstructured {
	var objectsInStatus = []*unstructured.Unstructured{}
	for _, o := range objStates {
		if o.Status == status {
			objectsInStatus = append(objectsInStatus, o.Object)
		}
	}
	return objectsInStatus
}

// FilterObjectStateNotInStatusType returns the objects not in the desired status.
func FilterObjectStateNotInStatusType(objStates ObjectStates, status StatusType) []*unstructured.Unstructured {
	var objectsNotInStatus = []*unstructured.Unstructured{}
	for _, o := range objStates {
		if o.Status != status {
			objectsNotInStatus = append(objectsNotInStatus, o.Object)
		}
	}
	return objectsNotInStatus
}

// Status returns the status for a given object and SatusReady
// by default for non-managed objects.
func Status(u *unstructured.Unstructured) (StatusType, error) {
	gk := u.GroupVersionKind().GroupKind()
	switch gk.String() {
	case "Pod":
		pod := &corev1.Pod{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, pod); err != nil {
			return StatusUnknown, err
		}
		return PodStatus(pod)
	case "Deployment.apps":
		dp := &appsv1.Deployment{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, dp); err != nil {
			return StatusUnknown, err
		}
		return DeploymentStatus(dp)
	case "ReplicaSet.apps":
		rs := &appsv1.ReplicaSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, rs); err != nil {
			return StatusUnknown, err
		}
		return ReplicasetStatus(rs)
	case "StatefulSet.apps":
		sts := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, sts); err != nil {
			return StatusUnknown, err
		}
		return StatefulSetStatus(sts)
	case "DaemonSet.apps":
		ds := &appsv1.DaemonSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ds); err != nil {
			return StatusUnknown, err
		}
		return DaemonSetStatus(ds)
	case "ReplicationController":
		rc := &corev1.ReplicationController{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, rc); err != nil {
			return StatusUnknown, err
		}

		return ReplicationControllerStatus(rc)
	default:
		return StatusReady, nil
	}
}

// PodStatus returns the status of a Pod.
func PodStatus(pod *corev1.Pod) (StatusType, error) {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && (condition.Reason == "PodCompleted" || condition.Status == corev1.ConditionTrue) {
			return StatusReady, nil
		}
	}
	return StatusInProgress, nil
}

// DeploymentStatus returns the status of a Deployment.
func DeploymentStatus(dp *appsv1.Deployment) (StatusType, error) {
	replicaFailure := false
	progressing := false
	available := false

	for _, condition := range dp.Status.Conditions {
		switch condition.Type {
		case appsv1.DeploymentProgressing:
			if condition.Status == corev1.ConditionTrue && condition.Reason == "NewReplicaSetAvailable" {
				progressing = true
			}
		case appsv1.DeploymentAvailable:
			if condition.Status == corev1.ConditionTrue {
				available = true
			}
		case appsv1.DeploymentReplicaFailure:
			if condition.Status == corev1.ConditionTrue {
				replicaFailure = true
				break
			}
		}
	}

	if dp.Status.ObservedGeneration == dp.Generation &&
		dp.Status.Replicas == *dp.Spec.Replicas &&
		dp.Status.ReadyReplicas == *dp.Spec.Replicas &&
		dp.Status.AvailableReplicas >= *dp.Spec.Replicas &&
		dp.Status.Conditions != nil && len(dp.Status.Conditions) > 0 &&
		(progressing || available) && !replicaFailure {

		return StatusReady, nil
	}

	return StatusInProgress, nil
}

// ReplicasetStatus returns the status of a ReplicasetStatus.
func ReplicasetStatus(rs *appsv1.ReplicaSet) (StatusType, error) {
	replicaFailure := false
	for _, condition := range rs.Status.Conditions {
		switch condition.Type {
		case appsv1.ReplicaSetReplicaFailure:
			if condition.Status == corev1.ConditionTrue {
				replicaFailure = true
				break
			}
		}
	}
	if rs.Status.ObservedGeneration == rs.Generation &&
		rs.Status.Replicas == *rs.Spec.Replicas &&
		rs.Status.ReadyReplicas >= *rs.Spec.Replicas &&
		rs.Status.AvailableReplicas == *rs.Spec.Replicas &&
		!replicaFailure {

		return StatusReady, nil
	}
	return StatusInProgress, nil
}

// StatefulSetStatus returns the status of a StatefulSet.
func StatefulSetStatus(sts *appsv1.StatefulSet) (StatusType, error) {
	if sts.Status.ObservedGeneration == sts.Generation &&
		sts.Status.Replicas == *sts.Spec.Replicas &&
		sts.Status.ReadyReplicas == *sts.Spec.Replicas &&
		sts.Status.CurrentReplicas == *sts.Spec.Replicas {

		return StatusReady, nil
	}
	return StatusInProgress, nil
}

// DaemonSetStatus returns the status of a DaemonSet.
func DaemonSetStatus(ds *appsv1.DaemonSet) (StatusType, error) {
	if ds.Status.ObservedGeneration == ds.Generation &&
		ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable &&
		ds.Status.DesiredNumberScheduled == ds.Status.NumberReady {

		return StatusReady, nil
	}
	return StatusInProgress, nil
}

// ReplicationControllerStatus returns the status of a ReplicationController.
func ReplicationControllerStatus(rc *corev1.ReplicationController) (StatusType, error) {
	replicaFailure := false
	for _, condition := range rc.Status.Conditions {
		switch condition.Type {
		case corev1.ReplicationControllerReplicaFailure:
			if condition.Status == corev1.ConditionTrue {
				replicaFailure = true
				break
			}
		}
	}

	if rc.Status.ObservedGeneration == rc.Generation &&
		rc.Status.Replicas == *rc.Spec.Replicas &&
		rc.Status.ReadyReplicas >= *rc.Spec.Replicas &&
		rc.Status.AvailableReplicas == *rc.Spec.Replicas &&
		!replicaFailure {

		return StatusReady, nil
	}
	return StatusInProgress, nil
}
