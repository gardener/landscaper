// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package podcache_test

import (
	"context"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/lock/podcache"
	"github.com/gardener/landscaper/test/utils/envtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("Test pod cache", func() {

	var (
		ctx   context.Context
		state *envtest.State
	)

	BeforeEach(func() {
		var err error

		ctx = logging.NewContext(context.Background(), logging.Discard())
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	It("test pod cache", func() {
		pod1, _ := createPod(ctx, "pod1", state)
		pod2, _ := createPod(ctx, "pod2", state)

		cache := podcache.NewPodCache(state.Client)

		exists, err := cache.PodExists(ctx, cl.ObjectKeyFromObject(pod1))
		Expect(exists).To(BeTrue())
		Expect(err).To(BeNil())

		exists, err = cache.PodExists(ctx, cl.ObjectKeyFromObject(pod2))
		Expect(exists).To(BeTrue())
		Expect(err).To(BeNil())

		pod3, _ := createPod(ctx, "pod3", state)
		exists, err = cache.PodExists(ctx, cl.ObjectKeyFromObject(pod3))
		Expect(exists).To(BeTrue())
		Expect(err).To(BeNil())

		exists, err = cache.PodExists(ctx, cl.ObjectKey{Namespace: state.Namespace, Name: "pod4"})
		Expect(exists).To(BeFalse())
		Expect(err).To(BeNil())

		_ = deletePod(ctx, pod3.Name, state)
		exists, err = cache.PodExists(ctx, cl.ObjectKeyFromObject(pod3))
		Expect(exists).To(BeTrue())
		Expect(err).To(BeNil())

		cache.TestSetLastUpdateTime(time.Now().Add(-1 * podcache.PodCacheTimeout).Add(-1 * time.Millisecond))
		exists, err = cache.PodExists(ctx, cl.ObjectKeyFromObject(pod3))
		Expect(exists).To(BeFalse())
		Expect(err).To(BeNil())
	})

})

func createPod(ctx context.Context, name string, state *envtest.State) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = state.Namespace
	pod.Spec.Containers = []corev1.Container{
		{
			Name:  "test",
			Image: "ubuntu",
		},
	}
	if err := state.Create(ctx, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

func deletePod(ctx context.Context, name string, state *envtest.State) error {
	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = state.Namespace
	pod.Spec.Containers = []corev1.Container{
		{
			Name:  "test",
			Image: "ubuntu",
		},
	}
	if err := state.Client.Delete(ctx, pod); err != nil {
		return err
	}

	return nil
}
