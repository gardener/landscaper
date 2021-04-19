// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitems

import (
	"context"
	"path"
	"time"

	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	gs "github.com/onsi/gomega/gstruct"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func ProgressingTimeoutTests(f *framework.Framework) {
	var (
		dumper      = f.Register()
		testdataDir = path.Join(f.RootPath, "test", "integration", "deployitems", "testdata")
	)

	ginkgo.Describe("Deploy Item Progressing Timeout", func() {

		var (
			ctx     context.Context
			state   *envtest.State
			cleanup framework.CleanupFunc
		)

		ginkgo.BeforeEach(func() {
			ctx = context.Background()
			var err error
			state, cleanup, err = f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
		})

		ginkgo.AfterEach(func() {
			defer ctx.Done()
			g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
		})

		ginkgo.It("should detect progressing timeouts", func() {
			ginkgo.By("create mock deploy item")
			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(testdataDir, "02-progressing-mock-di.yaml")))
			di.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, f.Client, di))

			progressingTimeout := di.Spec.Timeout.Duration

			ginkgo.By("waiting for timeout")
			time.Sleep(progressingTimeout)

			ginkgo.By("check for abort annotation and timestamp")
			g.Eventually(func() map[string]string {
				utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
				return di.Annotations
			}, waitingForReconcile, resyncTime).Should(gs.MatchKeys(gs.IgnoreExtras, gs.Keys{
				lsv1alpha1.AbortTimestampAnnotation: g.Not(g.Or(g.BeNil(), g.BeEmpty())),
				lsv1alpha1.OperationAnnotation:      g.BeEquivalentTo(lsv1alpha1.AbortOperation),
			}))
		})

		ginkgo.It("should not detect progressing timeouts for components in a final phase", func() {
			ginkgo.By("create mock deploy items")
			di1 := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di1, path.Join(testdataDir, "03-failed-mock-di.yaml")))
			di1.SetNamespace(state.Namespace)
			di2 := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di2, path.Join(testdataDir, "04-succeeded-mock-di.yaml")))
			di2.SetNamespace(state.Namespace)

			utils.ExpectNoError(state.Create(ctx, f.Client, di1))
			utils.ExpectNoError(state.Create(ctx, f.Client, di2))

			progressingTimeout := di1.Spec.Timeout.Duration
			tmp := di2.Spec.Timeout.Duration
			if tmp > progressingTimeout {
				progressingTimeout = tmp
			}

			ginkgo.By("waiting for timeout")
			time.Sleep(progressingTimeout + waitingForReconcile)
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di1.ObjectMeta), di1))
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di2.ObjectMeta), di2))

			ginkgo.By("check for abort annotation and timestamp")
			g.Expect(metav1.HasAnnotation(di1.ObjectMeta, string(lsv1alpha1helper.AbortTimestamp))).To(g.BeFalse(), "deploy item should not have an abort timestamp annotation")
			g.Expect(lsv1alpha1helper.HasOperation(di1.ObjectMeta, lsv1alpha1.AbortOperation)).To(g.BeFalse(), "deploy item should not have the abort operation annotation")
			g.Expect(metav1.HasAnnotation(di2.ObjectMeta, string(lsv1alpha1helper.AbortTimestamp))).To(g.BeFalse(), "deploy item should not have an abort timestamp annotation")
			g.Expect(lsv1alpha1helper.HasOperation(di2.ObjectMeta, lsv1alpha1.AbortOperation)).To(g.BeFalse(), "deploy item should not have the abort operation annotation")
		})

	})

}
