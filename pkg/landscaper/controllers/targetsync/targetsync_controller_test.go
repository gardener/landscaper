// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/clusters"
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
			ctrl = NewTargetSyncController(logging.Discard(), testenv.Client, clusters.NewTrivialSourceClientProvider(testenv.Client, nil))
		})

		AfterEach(func() {
			if state != nil {
				ctx := context.Background()
				defer ctx.Done()
				Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
				state = nil
			}
		})

		checkTarget := func(ctx context.Context, targetName, secretRefName, secretRefKey string) {
			target := &lsv1alpha1.Target{}
			target.Name = targetName
			target.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(target), target))

			Expect(target.Spec.SecretRef.Name).To(Equal(secretRefName))
			Expect(target.Spec.SecretRef.Key).To(Equal(secretRefKey))
			Expect(target.Spec.Type).To(Equal(targettypes.KubernetesClusterTargetType))
		}

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
			Expect(target.Spec.Type).To(Equal(targettypes.KubernetesClusterTargetType))
			Expect(secret.Data).To(Equal(sourceSecret.Data))
		}

		checkTargetAndSecretDoNotExist := func(ctx context.Context, secretName string) {
			Expect(wait.PollImmediate(time.Second, 10*time.Second, func() (done bool, err error) {

				target := &lsv1alpha1.Target{}
				target.Name = secretName
				target.Namespace = state.Namespace
				err1 := state.Client.Get(ctx, kutil.ObjectKeyFromObject(target), target)
				if err1 == nil {
					return false, nil
				} else if !errors.IsNotFound(err1) {
					return false, err1
				}

				secret := &corev1.Secret{}
				secret.Name = secretName
				secret.Namespace = state.Namespace
				err2 := state.Client.Get(ctx, kutil.ObjectKeyFromObject(target), target)
				if err2 == nil {
					return false, nil
				} else if !errors.IsNotFound(err2) {
					return false, err2
				}

				return true, nil

			})).To(Succeed())
		}

		It("should sync Secrets and react to their updates", func() {
			ctx := context.Background()

			const (
				targetSyncName   = "test-target-sync"
				secretName1      = "cluster1.kubeconfig"
				secretName2      = "cluster2.kubeconfig"
				sourceTargetName = "test-source-target-name"
			)

			var err error
			state, err = testenv.InitResourcesWithTwoNamespaces(ctx, "./testdata/state/test1")
			Expect(err).ToNot(HaveOccurred())

			tgs := &lsv1alpha1.TargetSync{}
			tgs.Name = targetSyncName
			tgs.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))
			tgs.Spec.CreateTargetToSource = true
			tgs.Spec.TargetToSourceName = sourceTargetName

			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))
			Expect(helper.HasOperation(tgs.ObjectMeta, lsv1alpha1.ReconcileOperation)).To(BeTrue())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecret(ctx, secretName1)
			checkTargetAndSecret(ctx, secretName2)
			checkTarget(ctx, sourceTargetName, tgs.Spec.SecretRef.Name, tgs.Spec.SecretRef.Key)

			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))
			Expect(helper.HasOperation(tgs.ObjectMeta, lsv1alpha1.ReconcileOperation)).To(BeFalse())

			// Update secret

			sourceSecret1 := &corev1.Secret{}
			sourceSecret1.Name = secretName1
			sourceSecret1.Namespace = state.Namespace2
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(sourceSecret1), sourceSecret1))
			sourceSecret1.StringData = map[string]string{"kubeconfig": "dummy-kubeconfig-updated"}
			testutils.ExpectNoError(state.Client.Update(ctx, sourceSecret1))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecret(ctx, secretName1)
			checkTargetAndSecret(ctx, secretName2)

			// Delete secret

			testutils.ExpectNoError(state.Client.Delete(ctx, sourceSecret1))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecretDoNotExist(ctx, secretName1)
			checkTargetAndSecret(ctx, secretName2)
		})

		It("should react to updates of the TargetSync object", func() {
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
			Expect(helper.HasOperation(tgs.ObjectMeta, lsv1alpha1.ReconcileOperation)).To(BeTrue())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecret(ctx, secretName1)
			checkTargetAndSecret(ctx, secretName2)
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))
			Expect(helper.HasOperation(tgs.ObjectMeta, lsv1alpha1.ReconcileOperation)).To(BeFalse())

			// Update TargetSync object

			tgs = &lsv1alpha1.TargetSync{}
			tgs.Name = targetSyncName
			tgs.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))

			tgs.Spec.SecretNameExpression = "^" + secretName1 + "$"
			testutils.ExpectNoError(state.Client.Update(ctx, tgs))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecret(ctx, secretName1)
			checkTargetAndSecretDoNotExist(ctx, secretName2)

			// Delete TargetSync object

			testutils.ExpectNoError(state.Client.Delete(ctx, tgs))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			checkTargetAndSecretDoNotExist(ctx, secretName1)
			checkTargetAndSecretDoNotExist(ctx, secretName2)
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
			Expect(helper.HasOperation(tgs1.ObjectMeta, lsv1alpha1.ReconcileOperation)).To(BeTrue())

			testutils.ShouldReconcileButRetry(ctx, ctrl, testutils.RequestFromObject(tgs1))

			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs1), tgs1))
			Expect(tgs1.Status.LastErrors).NotTo(BeEmpty())

			checkTargetAndSecretDoNotExist(ctx, secretName)
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs1), tgs1))
			Expect(helper.HasOperation(tgs1.ObjectMeta, lsv1alpha1.ReconcileOperation)).To(BeFalse())
		})
	})

})
