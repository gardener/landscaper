// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package healthcheck

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

type DefaultHealthCheck struct {
	Context          context.Context
	Client           client.Client
	Log              logr.Logger
	CurrentOp        string
	Timeout          *lsv1alpha1.Duration
	ManagedResources []lsv1alpha1.TypedObjectReference
}

// CheckResourcesHealth implements the default health check for Kubernetes manifests
func (d *DefaultHealthCheck) CheckResourcesHealth() error {

	if len(d.ManagedResources) == 0 {
		return nil
	}

	objects := make([]*unstructured.Unstructured, len(d.ManagedResources))
	for i, ref := range d.ManagedResources {
		obj := kutil.ObjectFromTypedObjectReference(&ref)
		objects[i] = obj
	}

	timeout := d.Timeout.Duration
	if err := WaitForObjectsHealthy(d.Context, timeout, d.Log, d.Client, objects, d.CheckObject); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			d.CurrentOp, "CheckResourceHealth", err.Error(), lsv1alpha1.ErrorHealthCheckTimeout)
	}

	return nil
}

// DefaultCheckObject checks if the object is healthy and returns an error otherwise.
// A non-managed object returns nil.
func (d *DefaultHealthCheck) CheckObject(u *unstructured.Unstructured) error {
	gk := u.GroupVersionKind().GroupKind()
	switch gk.String() {
	case "Pod":
		pod := &corev1.Pod{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, pod); err != nil {
			return err
		}
		return CheckPod(pod)
	case "Deployment.apps":
		dp := &appsv1.Deployment{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, dp); err != nil {
			return err
		}
		return CheckDeployment(dp)
	case "ReplicaSet.apps":
		rs := &appsv1.ReplicaSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, rs); err != nil {
			return err
		}
		return CheckReplicaSet(rs)
	case "StatefulSet.apps":
		sts := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, sts); err != nil {
			return err
		}
		return CheckStatefulSet(sts)
	case "DaemonSet.apps":
		ds := &appsv1.DaemonSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ds); err != nil {
			return err
		}
		return CheckDaemonSet(ds)
	case "ReplicationController":
		rc := &corev1.ReplicationController{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, rc); err != nil {
			return err
		}
		return CheckReplicationController(rc)
	default:
		return nil
	}
}

func outdatedGeneration(current, expected int64) error {
	return fmt.Errorf("observed generation outdated (%d/%d)", current, expected)
}

func notEnoughReadyReplicas(current, expected int32) error {
	return fmt.Errorf("not enough ready replicas (%d/%d)", current, expected)
}

func conditionInvalidStatus(conditionType string, expected, actual, reason, message string) error {
	return fmt.Errorf("condition %q has invalid status %s (expected %s) due to %s: %s",
		conditionType, actual, expected, reason, message)
}

func requiredConditionMissing(conditionType string) error {
	return fmt.Errorf("condition %q is missing", conditionType)
}

func checkConditionState(conditionType string, expected, actual, reason, message string) error {
	if expected != actual {
		return conditionInvalidStatus(conditionType, expected, actual, reason, message)
	}
	return nil
}

var (
	truePodConditionTypes = []corev1.PodConditionType{
		corev1.PodReady,
	}
)

func getPodCondition(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// CheckPod checks whether the given Pod is healthy.
// A Pod is considered healthy if it successfully completed
// or if it has the the PodReady condition set to true.
func CheckPod(pod *corev1.Pod) error {
	for _, trueConditionType := range truePodConditionTypes {
		conditionType := string(trueConditionType)
		condition := getPodCondition(pod.Status.Conditions, corev1.PodReady)
		if condition == nil {
			return requiredConditionMissing(conditionType)
		}
		if condition.Reason == "PodCompleted" {
			return nil
		}
		if err := checkConditionState(conditionType, string(corev1.ConditionTrue), string(condition.Status), condition.Reason, condition.Message); err != nil {
			return err
		}
	}

	return nil
}

var (
	trueDeploymentConditionTypes = []appsv1.DeploymentConditionType{
		appsv1.DeploymentAvailable,
	}

	trueOptionalDeploymentConditionTypes = []appsv1.DeploymentConditionType{
		appsv1.DeploymentProgressing,
	}

	falseOptionalDeploymentConditionTypes = []appsv1.DeploymentConditionType{
		appsv1.DeploymentReplicaFailure,
	}
)

func getDeploymentCondition(conditions []appsv1.DeploymentCondition, conditionType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// CheckDeployment checks whether the given Deployment is healthy.
// A Deployment is considered healthy if the controller observed its current revision and
// if the number of updated replicas is equal to the number of replicas.
func CheckDeployment(dp *appsv1.Deployment) error {
	if dp.Status.ObservedGeneration < dp.Generation {
		return outdatedGeneration(dp.Status.ObservedGeneration, dp.Generation)
	}

	for _, trueConditionType := range trueDeploymentConditionTypes {
		conditionType := string(trueConditionType)
		condition := getDeploymentCondition(dp.Status.Conditions, trueConditionType)
		if condition == nil {
			return requiredConditionMissing(conditionType)
		}
		if err := checkConditionState(conditionType, string(corev1.ConditionTrue), string(condition.Status), condition.Reason, condition.Message); err != nil {
			return err
		}
	}

	for _, trueOptionalConditionType := range trueOptionalDeploymentConditionTypes {
		conditionType := string(trueOptionalConditionType)
		condition := getDeploymentCondition(dp.Status.Conditions, trueOptionalConditionType)
		if condition == nil {
			continue
		}
		if err := checkConditionState(conditionType, string(corev1.ConditionTrue), string(condition.Status), condition.Reason, condition.Message); err != nil {
			return err
		}
	}

	for _, falseOptionalConditionType := range falseOptionalDeploymentConditionTypes {
		conditionType := string(falseOptionalConditionType)
		condition := getDeploymentCondition(dp.Status.Conditions, falseOptionalConditionType)
		if condition == nil {
			continue
		}
		if err := checkConditionState(conditionType, string(corev1.ConditionFalse), string(condition.Status), condition.Reason, condition.Message); err != nil {
			return err
		}
	}

	return nil
}

// CheckStatefulSet checks whether the given StatefulSet is healthy.
// A StatefulSet is considered healthy if its controller observed its current revision,
// it is not in an update (i.e. UpdateRevision is empty) and if its current replicas are equal to
// its desired replicas.
func CheckStatefulSet(sts *appsv1.StatefulSet) error {
	if sts.Status.ObservedGeneration < sts.Generation {
		return outdatedGeneration(sts.Status.ObservedGeneration, sts.Generation)
	}

	replicas := int32(1)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	if sts.Status.ReadyReplicas < replicas {
		return notEnoughReadyReplicas(sts.Status.ReadyReplicas, replicas)
	}
	return nil
}

func daemonSetMaxUnavailable(ds *appsv1.DaemonSet) int32 {
	if ds.Status.DesiredNumberScheduled == 0 || ds.Spec.UpdateStrategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
		return 0
	}

	rollingUpdate := ds.Spec.UpdateStrategy.RollingUpdate
	if rollingUpdate == nil {
		return 0
	}

	maxUnavailable, err := intstr.GetValueFromIntOrPercent(rollingUpdate.MaxUnavailable, int(ds.Status.DesiredNumberScheduled), false)
	if err != nil {
		return 0
	}

	return int32(maxUnavailable)
}

// CheckDaemonSet checks whether the given DaemonSet is healthy.
// A DaemonSet is considered healthy if its controller observed its current revision and if
// its desired number of scheduled pods is equal to its updated number of scheduled pods.
func CheckDaemonSet(ds *appsv1.DaemonSet) error {
	if ds.Status.ObservedGeneration < ds.Generation {
		return outdatedGeneration(ds.Status.ObservedGeneration, ds.Generation)
	}

	maxUnavailable := daemonSetMaxUnavailable(ds)

	if requiredAvailable := ds.Status.DesiredNumberScheduled - maxUnavailable; ds.Status.CurrentNumberScheduled < requiredAvailable {
		return notEnoughReadyReplicas(ds.Status.CurrentNumberScheduled, requiredAvailable)
	}
	return nil
}

// CheckReplicaSet checks whether the given ReplicaSet is healthy.
// A ReplicaSet is considered healthy if its controller observed its current revision and
// if the number of updated replicas is equal to the number of replicas.
func CheckReplicaSet(rs *appsv1.ReplicaSet) error {
	if rs.Status.ObservedGeneration < rs.Generation {
		return outdatedGeneration(rs.Status.ObservedGeneration, rs.Generation)
	}

	for _, condition := range rs.Status.Conditions {
		switch condition.Type {
		case appsv1.ReplicaSetReplicaFailure:
			conditionType := string(condition.Type)
			if err := checkConditionState(conditionType, string(corev1.ConditionFalse), string(condition.Status), condition.Reason, condition.Message); err != nil {
				return err
			}
		}
	}

	replicas := int32(1)
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}

	if rs.Status.ReadyReplicas < replicas {
		return notEnoughReadyReplicas(rs.Status.ReadyReplicas, replicas)
	}
	return nil
}

// CheckReplicationController checks whether the given ReplicationController is healthy.
// A ReplicationController is considered healthy if its controller observed its current revision and
// if the number of updated replicas is equal to the number of replicas.
func CheckReplicationController(rc *corev1.ReplicationController) error {
	if rc.Status.ObservedGeneration < rc.Generation {
		return outdatedGeneration(rc.Status.ObservedGeneration, rc.Generation)
	}

	for _, condition := range rc.Status.Conditions {
		switch condition.Type {
		case corev1.ReplicationControllerReplicaFailure:
			conditionType := string(condition.Type)
			if err := checkConditionState(conditionType, string(corev1.ConditionFalse), string(condition.Status), condition.Reason, condition.Message); err != nil {
				return err
			}
		}
	}

	replicas := int32(1)
	if rc.Spec.Replicas != nil {
		replicas = *rc.Spec.Replicas
	}

	if rc.Status.ReadyReplicas < replicas {
		return notEnoughReadyReplicas(rc.Status.ReadyReplicas, replicas)
	}
	return nil
}
