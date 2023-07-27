// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Retry handler", func() {

	Context("with clock", func() {

		var (
			op    *lsoperation.Operation
			ctrl  reconcile.Reconciler
			clok  *testing.FakePassiveClock
			state *envtest.State
		)

		BeforeEach(func() {
			var err error
			registryAccess, err := registries.GetFactory().NewLocalRegistryAccess("./testdata")
			Expect(err).ToNot(HaveOccurred())

			op = lsoperation.NewOperation(testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(registryAccess)

			clok = &testing.FakePassiveClock{}

			ctrl = installationsctl.NewTestActuator(*op, logging.Discard(), clok, &config.LandscaperConfiguration{
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

		It("should retry a failed installation", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test9")
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := state.Installations[state.Namespace+"/root"]

			t1 := time.Date(2020, time.May, 1, 8, 0, 0, 0, time.UTC)
			clok.SetTime(t1)

			// installation gets a new job id
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // removed
			Expect(inst.Status.JobID).NotTo(Equal(inst.Status.JobIDFinished))                                                              // new job id
			Expect(inst.Status.AutomaticReconcileStatus).To(BeNil())

			// installation fails because of missing target import; retry handler directly triggers a retry
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).To(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // added
			Expect(inst.ObjectMeta.Annotations).To(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // added
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil()) // initialized
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(1))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t1.UnixMilli()))

			// installation gets a new job id
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // removed
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // removed
			Expect(inst.Status.JobID).NotTo(Equal(inst.Status.JobIDFinished))                                                              // new job id
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(1))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t1.UnixMilli()))

			// reconcile fails; retry helper schedules next reconcile event
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation)))
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(1))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t1.UnixMilli()))

			// reconcile event that is too early for the next retry
			t2 := t1.Add(30 * time.Minute)
			clok.SetTime(t2)
			// retry handler does not add a reconcile annotation because it is too early
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation)))
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(1))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t1.UnixMilli()))

			// 2nd retry

			t3 := t1.Add(61 * time.Minute)
			clok.SetTime(t3)
			// retry helper adds reconcile annotation to trigger a retry
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).To(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // added
			Expect(inst.ObjectMeta.Annotations).To(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // added
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(2))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t3.UnixMilli()))

			// installation gets a new job id
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // removed
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // removed
			Expect(inst.Status.JobID).NotTo(Equal(inst.Status.JobIDFinished))                                                              // new job id
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(2))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t3.UnixMilli()))

			// reconcile fails; retry helper schedules next reconcile event
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation)))
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(2))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t3.UnixMilli()))

			// 3rd retry

			t4 := t3.Add(61 * time.Minute)
			clok.SetTime(t4)
			// retry helper adds reconcile annotation to trigger a retry
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).To(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // added
			Expect(inst.ObjectMeta.Annotations).To(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // added
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(3))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t4.UnixMilli()))

			// installation gets a new job id
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // removed
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // removed
			Expect(inst.Status.JobID).NotTo(Equal(inst.Status.JobIDFinished))                                                              // new job id
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(3))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t4.UnixMilli()))

			// reconcile fails; max number of retries reached
			testutils.ShouldReconcileButRetry(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation)))
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(3))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t4.UnixMilli()))

			// all retries done

			t5 := t4.Add(61 * time.Minute)
			clok.SetTime(t5)
			// retry handler does not add a reconcile annotation because the maximum number of retries is reached
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // not added
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // not added
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(3))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t4.UnixMilli()))

			// update installation and add reconcile annotation

			inst.Spec.Imports.Targets[0].Target = inst.Spec.Imports.Targets[0].Target + "-2"
			Expect(testutils.AddReconcileAnnotation(ctx, testenv, inst)).To(Succeed())

			t6 := t5.Add(3 * time.Hour)
			clok.SetTime(t6)
			// retry handler resets the retry status because of the reconcile annotation
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // removed
			Expect(inst.ObjectMeta.Annotations).NotTo(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // removed
			Expect(inst.Status.JobID).NotTo(Equal(inst.Status.JobIDFinished))                                                              // new job id
			Expect(inst.Status.AutomaticReconcileStatus).To(BeNil())                                                                       // reset

			// installation fails; retry handler directly triggers a retry
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).To(Succeed())
			Expect(inst.ObjectMeta.Annotations).To(HaveKeyWithValue(v1alpha1.OperationAnnotation, string(v1alpha1.ReconcileOperation))) // added
			Expect(inst.ObjectMeta.Annotations).To(HaveKey(v1alpha1.ReconcileReasonAnnotation))                                         // added
			Expect(inst.Status.JobID).To(Equal(inst.Status.JobIDFinished))
			Expect(inst.Status.AutomaticReconcileStatus).NotTo(BeNil())
			Expect(inst.Status.AutomaticReconcileStatus.Generation).To(Equal(inst.Generation))
			Expect(inst.Status.AutomaticReconcileStatus.NumberOfReconciles).To(Equal(1))
			Expect(inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time.UnixMilli()).To(Equal(t6.UnixMilli()))
		})
	})
})
