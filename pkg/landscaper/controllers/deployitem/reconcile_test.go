// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem_test

import (
	"context"
	"time"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	dictrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	"github.com/gardener/landscaper/test/utils"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Deploy Item Controller Reconcile Test", func() {

	var (
		state                                *envtest.State
		deployItemController, mockController reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error

		deployItemController, err = dictrl.NewController(testing.NullLogger{}, testenv.Client, api.LandscaperScheme, testPickupTimeoutDuration, testAbortingTimeoutDuration, testProgressingTimeoutDuration)
		Expect(err).ToNot(HaveOccurred())

		mockController, err = mockctlr.NewController(testing.NullLogger{}, testenv.Client, api.LandscaperScheme, &mockv1alpha1.Configuration{})
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

	It("Should detect pickup timeouts", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Prepare test deploy items")
		di := &lsv1alpha1.DeployItem{}
		diReq := testutils.Request("mock-di-prog", state.Namespace)
		// do not reconcile with mock deployer

		By("Set timed out reconcile timestamp annotation")
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		timedOut := metav1.Time{Time: time.Now().Add(-(testPickupTimeoutDuration.Duration + (5 * time.Second)))}
		metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.ReconcileTimestampAnnotation, timedOut.Format(time.RFC3339))
		utils.ExpectNoError(testenv.Client.Update(ctx, di))

		By("Verify that timed out deploy items are in 'Failed' phase")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status).To(MatchFields(IgnoreExtras, Fields{
			"Phase": Equal(lsv1alpha1.ExecutionPhaseFailed),
			"LastError": PointTo(MatchFields(IgnoreExtras, Fields{
				"Codes":  ContainElement(lsv1alpha1.ErrorTimeout),
				"Reason": Equal(dictrl.PickupTimeoutReason),
			})),
		}))
	})

	It("Should detect progressing timeouts", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Prepare test deploy items")
		di := &lsv1alpha1.DeployItem{}
		diReq := testutils.Request("mock-di-prog", state.Namespace)
		testutils.ShouldReconcile(ctx, mockController, diReq)
		testutils.ShouldReconcile(ctx, mockController, diReq)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
		Expect(di.Status.LastChangeReconcileTime).NotTo(BeNil())
		old := di.DeepCopy()

		// reconcile with deploy item controller should not do anything to the deploy item
		By("Verify that deploy item controller doesn't change anything if no timeout occurred")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di).To(Equal(old))

		By("Set timed out LastChangeReconcileTime timestamp")
		timedOut := metav1.Time{Time: time.Now().Add(-(testProgressingTimeoutDuration.Duration + (5 * time.Second)))}
		di.Status.LastChangeReconcileTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		By("Verify that timed out deploy items get an abort annotation")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Annotations).NotTo(BeNil())
		Expect(metav1.HasAnnotation(di.ObjectMeta, lsv1alpha1.AbortTimestampAnnotation)).To(BeTrue(), "deploy item should have an abort timestamp annotation")
		Expect(lsv1alpha1helper.HasOperation(di.ObjectMeta, lsv1alpha1.AbortOperation)).To(BeTrue(), "deploy item should have an abort operation annotation")
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
	})

	It("Should not detect progressing timeouts for deploy items in a final phase", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Prepare test deploy items")
		diS := &lsv1alpha1.DeployItem{}
		diF := &lsv1alpha1.DeployItem{}
		diReqS := testutils.Request("mock-di-succ", state.Namespace)
		diReqF := testutils.Request("mock-di-fail", state.Namespace)
		testutils.ShouldReconcile(ctx, mockController, diReqS)
		testutils.ShouldReconcile(ctx, mockController, diReqS)
		testutils.ShouldReconcile(ctx, mockController, diReqF)
		testutils.ShouldReconcile(ctx, mockController, diReqF)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReqS.NamespacedName, diS))
		utils.ExpectNoError(testenv.Client.Get(ctx, diReqF.NamespacedName, diF))
		Expect(diS.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(diF.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))
		Expect(diS.Status.LastChangeReconcileTime).NotTo(BeNil())
		Expect(diF.Status.LastChangeReconcileTime).NotTo(BeNil())

		By("Set timed out LastChangeReconcileTime timestamp")
		timedOut := metav1.Time{Time: time.Now().Add(-(testProgressingTimeoutDuration.Duration + (5 * time.Second)))}
		diS.Status.LastChangeReconcileTime = &timedOut
		diF.Status.LastChangeReconcileTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, diS))
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, diF))

		By("Verify that deploy items did not get an abort annotation")
		testutils.ShouldReconcile(ctx, deployItemController, diReqS)
		testutils.ShouldReconcile(ctx, deployItemController, diReqF)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReqS.NamespacedName, diS))
		utils.ExpectNoError(testenv.Client.Get(ctx, diReqF.NamespacedName, diF))
		Expect(diS.Annotations).To(BeNil())
		Expect(diF.Annotations).To(BeNil())
	})

	It("Should prefer a timeout specified in the deploy item over the default", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Prepare test deploy items")
		di := &lsv1alpha1.DeployItem{}
		diReq := testutils.Request("mock-di-prog-timeout", state.Namespace)
		testutils.ShouldReconcile(ctx, mockController, diReq)
		testutils.ShouldReconcile(ctx, mockController, diReq)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))

		By("Set timed out LastChangeReconcileTime timestamp (using default timeout duration)")
		timedOut := metav1.Time{Time: time.Now().Add(-(testProgressingTimeoutDuration.Duration + (5 * time.Second)))}
		di.Status.LastChangeReconcileTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		By("Verify that deploy item is not timed out")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Annotations).To(BeNil())

		By("Set timed out LastChangeReconcileTime timestamp (deploy item specific timeout duration)")
		timedOut = metav1.Time{Time: time.Now().Add(-(di.Spec.Timeout.Duration + (5 * time.Second)))}
		di.Status.LastChangeReconcileTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		By("Verify that deploy item is timed out")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Annotations).NotTo(BeNil())
		Expect(metav1.HasAnnotation(di.ObjectMeta, lsv1alpha1.AbortTimestampAnnotation)).To(BeTrue(), "deploy item should have an abort timestamp annotation")
		Expect(lsv1alpha1helper.HasOperation(di.ObjectMeta, lsv1alpha1.AbortOperation)).To(BeTrue(), "deploy item should have an abort operation annotation")
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
	})

	It("Should detect aborting timeouts", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Prepare test deploy items")
		di := &lsv1alpha1.DeployItem{}
		diReq := testutils.Request("mock-di-prog", state.Namespace)
		testutils.ShouldReconcile(ctx, mockController, diReq)
		testutils.ShouldReconcile(ctx, mockController, diReq)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))

		By("Set timed out abort timestamp annotation")
		timedOut := time.Now().Add(-(testAbortingTimeoutDuration.Duration + (5 * time.Second)))
		lsv1alpha1helper.SetOperation(&di.ObjectMeta, lsv1alpha1.AbortOperation)
		metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.AbortTimestampAnnotation, timedOut.Format(time.RFC3339))
		utils.ExpectNoError(testenv.Client.Update(ctx, di))

		By("Verify that timed out deploy items are in 'Failed' phase")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status).To(MatchFields(IgnoreExtras, Fields{
			"Phase": Equal(lsv1alpha1.ExecutionPhaseFailed),
			"LastError": PointTo(MatchFields(IgnoreExtras, Fields{
				"Codes":  ContainElement(lsv1alpha1.ErrorTimeout),
				"Reason": Equal(dictrl.AbortingTimeoutReason),
			})),
		}))
	})

})
