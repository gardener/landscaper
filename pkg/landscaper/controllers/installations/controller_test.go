// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Installation Controller", func() {

	Context("reconcile", func() {
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

			registry := cnudie.NewRegistry(fakeCompRepo)
			op = lsoperation.NewOperation(testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(registry)

			ctrl = installationsctl.NewTestActuator(*op, logging.Discard(), clock.RealClock{}, &config.LandscaperConfiguration{
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

		It("should not reconcile an installation without reconcile annotation", func() {
			// We consider an Installation in a finished state.
			// The Installation has no reconcile annotation. Therefore, a reconciliation should have no effect.
			// After the reconciliation, the jobID and jobIDFinished should be the same as before.
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test6")
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			inst.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			jobID := inst.Status.JobID
			Expect(inst.Status.JobIDFinished).To(Equal(jobID))
			Expect(inst.Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Succeeded))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.Status.JobID).To(Equal(jobID))
			Expect(inst.Status.JobIDFinished).To(Equal(jobID))
			Expect(inst.Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Succeeded))
		})

		It("should pass an interrupt annotation to an execution", func() {
			// We consider an unfinished Installation with an Execution and a subinstallation.
			// The Installation has an interrupt annotation. After a reconciliation the annotation should be
			// added to the Execution and subinstallation, and it should be removed from the root Installation.
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test7")
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			inst.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.ObjectMeta.Annotations).To(HaveKeyWithValue(lsv1alpha1.OperationAnnotation, string(lsv1alpha1.InterruptOperation)))

			exec := &lsv1alpha1.Execution{}
			exec.Name = inst.Status.ExecutionReference.Name
			exec.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(lsv1alpha1.OperationAnnotation, string(lsv1alpha1.InterruptOperation)))

			subinst := &lsv1alpha1.Installation{}
			subinst.Name = "subinst"
			subinst.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst))
			Expect(subinst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(lsv1alpha1.OperationAnnotation, string(lsv1alpha1.InterruptOperation)))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(lsv1alpha1.OperationAnnotation, string(lsv1alpha1.InterruptOperation)))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.ObjectMeta.Annotations).To(HaveKeyWithValue(lsv1alpha1.OperationAnnotation, string(lsv1alpha1.InterruptOperation)))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst))
			Expect(subinst.ObjectMeta.Annotations).To(HaveKeyWithValue(lsv1alpha1.OperationAnnotation, string(lsv1alpha1.InterruptOperation)))
		})
	})

})
