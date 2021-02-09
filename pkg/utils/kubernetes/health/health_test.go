// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package health_test

import (
	"github.com/gardener/landscaper/pkg/utils/kubernetes/health"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func replicas(i int32) *int32 {
	return &i
}

var _ = Describe("health", func() {
	Describe("CheckPod", func() {
		DescribeTable("pod",
			func(pod *corev1.Pod, matcher types.GomegaMatcher) {
				err := health.CheckPod(pod)
				Expect(err).To(matcher)
			},
			Entry("healthy running", &corev1.Pod{
				Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				}},
			}, BeNil()),
			Entry("healthy completed", &corev1.Pod{
				Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Reason: "PodCompleted",
						Status: corev1.ConditionFalse,
					},
				}},
			}, BeNil()),
			Entry("not ready", &corev1.Pod{
				Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				}},
			}, HaveOccurred()),
			Entry("missing conditions", &corev1.Pod{
				Status: corev1.PodStatus{Conditions: []corev1.PodCondition{}},
			}, HaveOccurred()),
		)
	})

	Describe("CheckDeployment", func() {
		DescribeTable("deployments",
			func(deployment *appsv1.Deployment, matcher types.GomegaMatcher) {
				err := health.CheckDeployment(deployment)
				Expect(err).To(matcher)
			},
			Entry("healthy", &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
				}},
			}, BeNil()),
			Entry("healthy with progressing", &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   appsv1.DeploymentProgressing,
						Status: corev1.ConditionTrue,
					},
				}},
			}, BeNil()),
			Entry("not observed at latest version", &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			}, HaveOccurred()),
			Entry("not available", &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   appsv1.DeploymentProgressing,
						Status: corev1.ConditionTrue,
					},
				}},
			}, HaveOccurred()),
			Entry("not progressing", &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   appsv1.DeploymentProgressing,
						Status: corev1.ConditionFalse,
					},
				}},
			}, HaveOccurred()),
			Entry("replica failure", &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   appsv1.DeploymentReplicaFailure,
						Status: corev1.ConditionTrue,
					},
				}},
			}, HaveOccurred()),
			Entry("available | progressing missing", &appsv1.Deployment{}, HaveOccurred()),
		)
	})

	Describe("CheckStatefulSet", func() {
		DescribeTable("statefulsets",
			func(statefulSet *appsv1.StatefulSet, matcher types.GomegaMatcher) {
				err := health.CheckStatefulSet(statefulSet)
				Expect(err).To(matcher)
			},
			Entry("healthy", &appsv1.StatefulSet{
				Spec:   appsv1.StatefulSetSpec{Replicas: replicas(1)},
				Status: appsv1.StatefulSetStatus{CurrentReplicas: 1, ReadyReplicas: 1},
			}, BeNil()),
			Entry("healthy with nil replicas", &appsv1.StatefulSet{
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
			}, BeNil()),
			Entry("not observed at latest version", &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			}, HaveOccurred()),
			Entry("empty status", &appsv1.StatefulSet{
				Status: appsv1.StatefulSetStatus{},
			}, HaveOccurred()),
			Entry("not enough ready replicas", &appsv1.StatefulSet{
				Spec:   appsv1.StatefulSetSpec{Replicas: replicas(2)},
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
			}, HaveOccurred()),
		)
	})

	Describe("CheckDaemonSet", func() {
		oneUnavailable := intstr.FromInt(1)
		DescribeTable("daemonsets",
			func(daemonSet *appsv1.DaemonSet, matcher types.GomegaMatcher) {
				err := health.CheckDaemonSet(daemonSet)
				Expect(err).To(matcher)
			},
			Entry("healthy", &appsv1.DaemonSet{}, BeNil()),
			Entry("healthy with one unavailable", &appsv1.DaemonSet{
				Spec: appsv1.DaemonSetSpec{UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
					Type: appsv1.RollingUpdateDaemonSetStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDaemonSet{
						MaxUnavailable: &oneUnavailable,
					},
				}},
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 2,
					CurrentNumberScheduled: 1,
				},
			}, BeNil()),
			Entry("not observed at latest version", &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			}, HaveOccurred()),
			Entry("empty status", &appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{},
			}, BeNil()),
			Entry("not enough updated scheduled", &appsv1.DaemonSet{
				Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 1},
			}, HaveOccurred()),
		)
	})

	Describe("CheckReplicaSet", func() {
		DescribeTable("replicaSet",
			func(replicaSet *appsv1.ReplicaSet, matcher types.GomegaMatcher) {
				err := health.CheckReplicaSet(replicaSet)
				Expect(err).To(matcher)
			},
			Entry("healthy", &appsv1.ReplicaSet{
				Spec:   appsv1.ReplicaSetSpec{Replicas: replicas(1)},
				Status: appsv1.ReplicaSetStatus{Replicas: 1, ReadyReplicas: 1},
			}, BeNil()),
			Entry("healthy with nil replicas", &appsv1.ReplicaSet{
				Status: appsv1.ReplicaSetStatus{ReadyReplicas: 1},
			}, BeNil()),
			Entry("empty status", &appsv1.ReplicaSet{
				Status: appsv1.ReplicaSetStatus{},
			}, HaveOccurred()),
			Entry("replica failure", &appsv1.ReplicaSet{
				Status: appsv1.ReplicaSetStatus{Conditions: []appsv1.ReplicaSetCondition{
					{
						Type:   appsv1.ReplicaSetReplicaFailure,
						Status: corev1.ConditionTrue,
					},
				}},
			}, HaveOccurred()),
			Entry("not observed at latest version", &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			}, HaveOccurred()),
			Entry("not enough ready replicas", &appsv1.ReplicaSet{
				Spec:   appsv1.ReplicaSetSpec{Replicas: replicas(2)},
				Status: appsv1.ReplicaSetStatus{ReadyReplicas: 1},
			}, HaveOccurred()),
		)
	})

	Describe("CheckReplicationController", func() {
		DescribeTable("replicationController",
			func(replicationController *corev1.ReplicationController, matcher types.GomegaMatcher) {
				err := health.CheckReplicationController(replicationController)
				Expect(err).To(matcher)
			},
			Entry("healthy", &corev1.ReplicationController{
				Spec:   corev1.ReplicationControllerSpec{Replicas: replicas(1)},
				Status: corev1.ReplicationControllerStatus{Replicas: 1, ReadyReplicas: 1},
			}, BeNil()),
			Entry("healthy with nil replicas", &corev1.ReplicationController{
				Status: corev1.ReplicationControllerStatus{ReadyReplicas: 1},
			}, BeNil()),
			Entry("not observed at latest version", &corev1.ReplicationController{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			}, HaveOccurred()),
			Entry("empty status", &corev1.ReplicationController{
				Status: corev1.ReplicationControllerStatus{},
			}, HaveOccurred()),
			Entry("replica failure", &corev1.ReplicationController{
				Status: corev1.ReplicationControllerStatus{Conditions: []corev1.ReplicationControllerCondition{
					{
						Type:   corev1.ReplicationControllerReplicaFailure,
						Status: corev1.ConditionTrue,
					},
				}},
			}, HaveOccurred()),
			Entry("not enough ready replicas", &corev1.ReplicationController{
				Spec:   corev1.ReplicationControllerSpec{Replicas: replicas(2)},
				Status: corev1.ReplicationControllerStatus{ReadyReplicas: 1},
			}, HaveOccurred()),
		)
	})
})
