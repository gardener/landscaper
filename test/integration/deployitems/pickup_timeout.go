// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitems

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	gs "github.com/onsi/gomega/gstruct"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func PickupTimeoutTests(f *framework.Framework) {
	ginkgo.Describe("Deploy Item Pickup Timeout", func() {
		var (
			dumper      = f.Register()
			testdataDir = path.Join(f.RootPath, "test", "integration", "deployitems", "testdata")

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

		ginkgo.It("should detect pickup timeouts", func() {
			ginkgo.By("create dummy installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "00-dummy-installation.yaml")))
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			ginkgo.By("verify that deploy items have been created")
			di := &lsv1alpha1.DeployItem{}
			g.Eventually(func() error {
				err := f.Client.Get(ctx, namespacedName(inst.ObjectMeta), inst)
				if err != nil {
					return err
				}
				if inst.Status.ExecutionReference == nil {
					return errors.New("no execution reference")
				}
				exec := &lsv1alpha1.Execution{}
				err = f.Client.Get(ctx, inst.Status.ExecutionReference.NamespacedName(), exec)
				if err != nil {
					return err
				}
				if exec.Status.DeployItemReferences == nil || len(exec.Status.DeployItemReferences) == 0 {
					return errors.New("no deployment references defined")
				}
				err = f.Client.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), di)
				if err != nil {
					return err
				}
				return nil
			}, waitingForDeployItems, resyncTime).Should(g.Succeed(), "unable to fetch deploy item")

			ginkgo.By("check for timestamp annotation")
			// checking whether the set timestamp is up-to-date is difficult due to potential differences between the
			// system times of the machine running the landscaper and the machine running the tests
			// so just check for existence of the annotation
			g.Expect(lsv1alpha1helper.HasTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp)).To(g.BeTrue(), "deploy item should have a reconcile timestamp annotation")

			ginkgo.By("check for pickup timeout")
			time.Sleep(deployItemPickupTimeout)
			g.Eventually(func() lsv1alpha1.DeployItemStatus {
				utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
				return di.Status
			}, waitingForReconcile, resyncTime).Should(gs.MatchFields(gs.IgnoreExtras, gs.Fields{
				"Phase": g.Equal(lsv1alpha1.ExecutionPhaseFailed),
				"LastError": gs.PointTo(gs.MatchFields(gs.IgnoreExtras, gs.Fields{
					"Codes":  g.ContainElement(lsv1alpha1.ErrorTimeout),
					"Reason": g.Equal(deployitem.PickupTimeoutReason),
				})),
			}))
		})

		ginkgo.It("should not detect pickup timeouts for components with working deployers", func() {
			ginkgo.By("create mock installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "01-mock-installation.yaml")))
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			ginkgo.By("verify that deploy items have been created")
			di := &lsv1alpha1.DeployItem{}
			g.Eventually(func() (*lsv1alpha1.DeployItem, error) {
				err := f.Client.Get(ctx, namespacedName(inst.ObjectMeta), inst)
				if err != nil || inst.Status.ExecutionReference == nil {
					return nil, err
				}
				exec := &lsv1alpha1.Execution{}
				err = f.Client.Get(ctx, inst.Status.ExecutionReference.NamespacedName(), exec)
				if err != nil || exec.Status.DeployItemReferences == nil || len(exec.Status.DeployItemReferences) == 0 {
					return nil, err
				}
				err = f.Client.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), di)
				if err != nil {
					return nil, err
				}
				return di, err
			}, waitingForDeployItems, resyncTime).ShouldNot(g.BeNil(), "unable to fetch deploy item")

			ginkgo.By("verify that deploy item is not timed out")
			time.Sleep(deployItemPickupTimeout + waitingForReconcile) // wait for a potential timeout to happen
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
			// check that deploy item does not have a pickup timeout
			g.Expect(di.Status.LastError).To(g.Or(g.BeNil(), gs.MatchFields(gs.IgnoreExtras, gs.Fields{
				"Codes":  g.Not(g.ContainElement(lsv1alpha1.ErrorTimeout)),
				"Reason": g.Not(g.Equal(deployitem.PickupTimeoutReason)),
			})))
		})

	})

}
