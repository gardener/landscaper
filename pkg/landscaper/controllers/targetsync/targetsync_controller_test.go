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

		It("should sync targets", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResourcesWithTwoNamespaces(ctx, "./testdata/state/test1")
			Expect(err).ToNot(HaveOccurred())

			tgs := &lsv1alpha1.TargetSync{}
			tgs.Name = "test-target-sync"
			tgs.Namespace = state.Namespace
			testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(tgs), tgs))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(tgs))

			secretNames := []string{"cluster1.kubeconfig", "cluster2.kubeconfig"}
			for _, secretName := range secretNames {
				target := &lsv1alpha1.Target{}
				target.Name = secretName
				target.Namespace = state.Namespace
				testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(target), target))

				secret := &corev1.Secret{}
				secret.Name = secretName
				secret.Namespace = state.Namespace
				testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(secret), secret))
			}
		})
	})

})
