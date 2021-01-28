// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SetReadyStatus sets the object status to Ready for workload type resources.
func SetReadyStatus(ctx context.Context, c client.Client, u *unstructured.Unstructured) error {
	gk := u.GroupVersionKind().GroupKind()
	switch gk.String() {
	case "Pod":
		pod := &corev1.Pod{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, pod); err != nil {
			return err
		}
		return SetPodReady(ctx, c, pod)
	case "StatefulSet.apps":
		sts := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, sts); err != nil {
			return err
		}
		return SetStatefulSetReady(ctx, c, sts)
	case "Deployment.apps":
		dp := &appsv1.Deployment{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, dp); err != nil {
			return err
		}
		return SetDeploymentReady(ctx, c, dp)
	case "ReplicaSet.apps":
		rs := &appsv1.ReplicaSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, rs); err != nil {
			return err
		}
		return SetReplicaSetReady(ctx, c, rs)
	case "DaemonSet.apps":
		ds := &appsv1.DaemonSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ds); err != nil {
			return err
		}
		return SetDaemonSetReady(ctx, c, ds)
	case "ReplicationController":
		rc := &corev1.ReplicationController{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, rc); err != nil {
			return err
		}
		return SetReplicationControllerReady(ctx, c, rc)
	default:
		return nil
	}
}

// SetPodReady sets a Pod status to Ready.
func SetPodReady(ctx context.Context, client client.Client, pod *corev1.Pod) error {
	status := corev1.PodStatus{
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	if _, err := controllerutil.CreateOrPatch(ctx, client, pod, func() error {
		pod.Status = status
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// SetDeploymentReady sets a Deployment status to Ready.
func SetDeploymentReady(ctx context.Context, client client.Client, dp *appsv1.Deployment) error {
	status := appsv1.DeploymentStatus{
		ObservedGeneration: dp.Generation,
		Replicas:           *dp.Spec.Replicas,
		ReadyReplicas:      *dp.Spec.Replicas,
		AvailableReplicas:  *dp.Spec.Replicas,
		Conditions: []appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionTrue,
			},
		},
	}

	if _, err := controllerutil.CreateOrPatch(ctx, client, dp, func() error {
		dp.Status = status
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// SetReplicaSetReady sets a ReplicaSet status to Ready.
func SetReplicaSetReady(ctx context.Context, client client.Client, rs *appsv1.ReplicaSet) error {
	status := appsv1.ReplicaSetStatus{
		ObservedGeneration: rs.Generation,
		Replicas:           *rs.Spec.Replicas,
		ReadyReplicas:      *rs.Spec.Replicas,
		AvailableReplicas:  *rs.Spec.Replicas,
	}

	if _, err := controllerutil.CreateOrPatch(ctx, client, rs, func() error {
		rs.Status = status
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// SetStatefulSetReady sets a StatefulSet status to Ready.
func SetStatefulSetReady(ctx context.Context, client client.Client, sts *appsv1.StatefulSet) error {
	status := appsv1.StatefulSetStatus{
		ObservedGeneration: sts.Generation,
		Replicas:           *sts.Spec.Replicas,
		ReadyReplicas:      *sts.Spec.Replicas,
		CurrentReplicas:    *sts.Spec.Replicas,
	}

	if _, err := controllerutil.CreateOrPatch(ctx, client, sts, func() error {
		sts.Status = status
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// SetDaemonSetReady sets a DaemonSet status to Ready.
func SetDaemonSetReady(ctx context.Context, client client.Client, ds *appsv1.DaemonSet) error {
	status := appsv1.DaemonSetStatus{
		ObservedGeneration: ds.Generation,
		// NumberAvailable is set in the status when pods are actually running
		// on actual nodes with kubelet so we set an arbitrary value.
		NumberAvailable:        1,
		DesiredNumberScheduled: 1,
	}

	if _, err := controllerutil.CreateOrPatch(ctx, client, ds, func() error {
		ds.Status = status
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// SetReplicationControllerReady sets a ReplicationController status to Ready.
func SetReplicationControllerReady(ctx context.Context, client client.Client, rc *corev1.ReplicationController) error {
	status := corev1.ReplicationControllerStatus{
		ObservedGeneration: rc.Generation,
		Replicas:           *rc.Spec.Replicas,
		ReadyReplicas:      *rc.Spec.Replicas,
		AvailableReplicas:  *rc.Spec.Replicas,
	}

	if _, err := controllerutil.CreateOrPatch(ctx, client, rc, func() error {
		rc.Status = status
		return nil
	}); err != nil {
		return err
	}
	return nil
}
