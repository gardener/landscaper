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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func AbortingTimeoutTests(f *framework.Framework) {
	var (
		dumper      = f.Register()
		testdataDir = path.Join(f.RootPath, "test", "integration", "deployitems", "testdata")
	)

	ginkgo.Describe("Deploy Item Aborting Timeout", func() {

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

		ginkgo.It("should detect aborting timeouts", func() {
			ginkgo.By("create mock deploy item")
			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(testdataDir, "02-progressing-mock-di.yaml")))
			di.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, f.Client, di))

			ginkgo.By("abort deploy item")
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
			g.Eventually(func() error { // single update operation seems to be flakey
				lsv1alpha1helper.SetAbortOperationAndTimestamp(&di.ObjectMeta)
				return f.Client.Update(ctx, di)
			}, retryUpdates, resyncTime).ShouldNot(g.HaveOccurred())

			ginkgo.By("waiting for timeout")
			time.Sleep(deployItemAbortingTimeout)

			ginkgo.By("check for aborting timeout")
			g.Eventually(func() lsv1alpha1.DeployItemStatus {
				utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
				return di.Status
			}, waitingForReconcile, resyncTime).Should(gs.MatchFields(gs.IgnoreExtras, gs.Fields{
				"Phase": g.Equal(lsv1alpha1.ExecutionPhaseFailed),
				"LastError": gs.PointTo(gs.MatchFields(gs.IgnoreExtras, gs.Fields{
					"Codes":  g.ContainElement(lsv1alpha1.ErrorTimeout),
					"Reason": g.Equal(deployitem.AbortingTimeoutReason),
				})),
			}))
		})
	})

}
