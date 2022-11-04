// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("TargetSync Controller", func() {

	Context("reconcile", func() {
		var (
			ctrl  reconcile.Reconciler
			state *envtest.State
		)

		BeforeEach(func() {
			var err error
			ctrl, err = NewTargetSyncController(logging.Discard(), testenv.Client, NewTrivialSourceClientProvider(testenv.Client))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if state != nil {
				ctx := context.Background()
				defer ctx.Done()
				Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
				state = nil
			}
		})

		checkTargetAndSecret := func(ctx context.Context, secretName string) {
			sourceSecret := &corev1.Secret{}
			sourceSecret.Name = secretName
			sourceSecret.Namespace = state.Namespace2
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(sourceSecret), sourceSecret))

			target := &lsv1alpha1.Target{}
			target.Name = secretName
			target.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(target), target))

			secret := &corev1.Secret{}
			secret.Name = secretName
			secret.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(secret), secret))

			Expect(target.Spec.SecretRef.Name).To(Equal(secret.Name))
			Expect(target.Spec.SecretRef.Namespace).To(Equal(secret.Namespace))
			Expect(target.Spec.Type).To(Equal(lsv1alpha1.KubernetesClusterTargetType))
			Expect(secret.Data).To(Equal(sourceSecret.Data))
		}

		checkTargetAndSecretDoNotExist := func(ctx context.Context, secretName string) {
			target := &lsv1alpha1.Target{}
			target.Name = secretName
			target.Namespace = state.Namespace
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(target), target)).NotTo(Succeed())

			secret := &corev1.Secret{}
			secret.Name = secretName
			secret.Namespace = state.Namespace
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(secret), secret)).NotTo(Succeed())
		}

		It("should sync targets", func() {
			ctx := context.Background()

			const (
				targetSyncName = "test-target-sync"
				secretName1    = "cluster1.kubeconfig"
				secretName2    = "cluster2.kubeconfig"
			)

			var err error
			state, err = testenv.InitResourcesWithTwoNamespaces(ctx, "./testdata/state/test1")
			Expect(err).ToNot(HaveOccurred())

			tgs := &lsv1alpha1.TargetSync{}
			tgs.Name = targetSyncName
			tgs.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			for _, secretName := range []string{secretName1, secretName2} {
				checkTargetAndSecret(ctx, secretName)
			}

			// Update secret

			sourceSecret1 := &corev1.Secret{}
			sourceSecret1.Name = secretName1
			sourceSecret1.Namespace = state.Namespace2
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(sourceSecret1), sourceSecret1))
			sourceSecret1.StringData = map[string]string{"kubeconfig": "dummy-kubeconfig-updated"}
			testutils.ExpectNoError(state.Client.Update(ctx, sourceSecret1))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			for _, secretName := range []string{secretName1, secretName2} {
				checkTargetAndSecret(ctx, secretName)
			}

			// Delete secret

			testutils.ExpectNoError(state.Client.Delete(ctx, sourceSecret1))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecretDoNotExist(ctx, secretName1)
			checkTargetAndSecret(ctx, secretName2)
		})

		It("should not sync if there is more than one TargetSync object", func() {
			ctx := context.Background()

			const (
				targetSyncName1 = "test-target-sync-1"
				secretName      = "cluster.kubeconfig"
			)

			var err error
			state, err = testenv.InitResourcesWithTwoNamespaces(ctx, "./testdata/state/test2")
			Expect(err).ToNot(HaveOccurred())

			tgs1 := &lsv1alpha1.TargetSync{}
			tgs1.Name = targetSyncName1
			tgs1.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs1), tgs1))

			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(tgs1))

			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs1), tgs1))
			Expect(tgs1.Status.LastErrors).NotTo(BeEmpty())

			checkTargetAndSecretDoNotExist(ctx, secretName)
		})
	})

})
