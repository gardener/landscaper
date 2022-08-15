// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	"github.com/gardener/component-spec/bindings-go/ctf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Reconcile", func() {

	Context("PhasePropagation", func() {

		var (
			op   *lsoperation.Operation
			ctrl reconcile.Reconciler

			state        *envtest.State
			fakeCompRepo ctf.ComponentResolver
		)

		BeforeEach(func() {
			var err error
			fakeCompRepo, err = componentsregistry.NewLocalClient("./testdata")
			Expect(err).ToNot(HaveOccurred())

			op = lsoperation.NewOperation(testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(fakeCompRepo)

			ctrl = installationsctl.NewTestActuator(*op, logging.Discard(), &config.LandscaperConfiguration{
				Registry: config.RegistryConfiguration{
					Local: &config.LocalRegistryConfiguration{
						RootPath: "./testdata",
					},
				},
			})
		})

		AfterEach(func() {
			if state != nil {
				ctx := context.Background()
				defer ctx.Done()
				Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
				state = nil
			}
		})

		It("should propagate phase changes from executions to installations", func() {
			if utils.IsNewReconcile() {
				// propagating changes upwards is currently not supported by the new reconcile
				return
			}

			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test4")
			testutils.ExpectNoError(err)
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := state.Installations[state.Namespace+"/root"]
			exec := state.Executions[state.Namespace+"/subexec"]

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))

			// set execution phase to 'Failed' and check again
			exec.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, exec))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseFailed))

			// set execution phase to 'Succeeded' and check again
			exec.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, exec))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))
		})

		It("should propagate phase changes from subinstallations to installations", func() {
			if utils.IsNewReconcile() {
				// propagating changes upwards is currently not supported by the new reconcile
				return
			}

			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test4")
			testutils.ExpectNoError(err)
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := state.Installations[state.Namespace+"/root"]
			subinst := state.Installations[state.Namespace+"/subinst"]

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))

			// set subinstallation phase to 'Failed' and check again
			subinst.Status.Phase = lsv1alpha1.ComponentPhaseFailed
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, subinst))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseFailed))

			// set subinstallation phase to 'Succeeded' and check again
			subinst.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, subinst))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))
		})

		It("should trigger reconcile of execution", func() {
			if utils.IsNewReconcile() {
				// this test is not relevant anymore are the whole strategy has changed
				return
			}

			// A reconciliation of a failed installation with reconcile annotation should trigger a
			// reconciliation of its failed execution by changing the ReconcileID in the execution spec.
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test5")
			testutils.ExpectNoError(err)
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := state.Installations[state.Namespace+"/root"]
			subinst := state.Installations[state.Namespace+"/subinst"]
			exec := state.Executions[state.Namespace+"/root"]
			oldReconcileID := exec.Spec.ReconcileID

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseProgressing))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst))
			lsv1alpha1helper.HasOperation(subinst.ObjectMeta, lsv1alpha1.ReconcileOperation)

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.Spec.ReconcileID).NotTo(Equal(oldReconcileID))
		})
	})
})
