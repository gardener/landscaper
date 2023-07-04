// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	dictrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	utils2 "github.com/gardener/landscaper/pkg/utils"
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

		deployItemController, err = dictrl.NewController(logging.Discard(), testenv.Client, api.LandscaperScheme,
			&testPickupTimeoutDuration, &testProgressingTimeoutDuration, 1000)
		Expect(err).ToNot(HaveOccurred())

		mockController, err = mockctlr.NewController(logging.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024), mockv1alpha1.Configuration{})
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

		Expect(testutils.UpdateJobIdForDeployItem(ctx, testenv, di, timedOut)).ToNot(HaveOccurred())

		By("Verify that timed out deploy items are in 'Failed' phase")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed))
		Expect(utils2.IsDeployItemJobIDsIdentical(di)).To(BeTrue())
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
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(testenv.Client.Get(ctx, testutils.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutils.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

		testutils.ShouldReconcile(ctx, mockController, diReq)
		testutils.ShouldReconcile(ctx, mockController, diReq)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(utils2.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Progressing)).To(BeTrue())
		Expect(utils2.IsDeployItemJobIDsIdentical(di)).To(BeFalse())
		Expect(di.Status.LastReconcileTime).NotTo(BeNil())
		old := di.DeepCopy()

		// reconcile with deploy item controller should not do anything to the deploy item
		By("Verify that deploy item controller doesn't change anything if no timeout occurred")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di).To(Equal(old))

		By("Set timed out LastReconcileTime timestamp")
		timedOut := metav1.Time{Time: time.Now().Add(-(testProgressingTimeoutDuration.Duration + (5 * time.Second)))}
		di.Status.LastReconcileTime = &timedOut
		di.Status.JobIDGenerationTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		By("Verify that timed out deploy items fail")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(utils2.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Failed)).To(BeTrue())
		Expect(di.Status.LastError.Reason).To(Equal(lsv1alpha1.ProgressingTimeoutReason))
		Expect(di.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
		Expect(utils2.IsDeployItemJobIDsIdentical(di)).To(BeTrue())
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

		utils.ExpectNoError(testenv.Client.Get(ctx, diReqS.NamespacedName, diS))
		Expect(testenv.Client.Get(ctx, testutils.Request(diS.GetName(), diS.GetNamespace()).NamespacedName, diS)).To(Succeed())
		Expect(testutils.UpdateJobIdForDeployItem(ctx, testenv, diS, metav1.Now())).ToNot(HaveOccurred())

		utils.ExpectNoError(testenv.Client.Get(ctx, diReqF.NamespacedName, diF))
		Expect(testenv.Client.Get(ctx, testutils.Request(diF.GetName(), diF.GetNamespace()).NamespacedName, diF)).To(Succeed())
		Expect(testutils.UpdateJobIdForDeployItem(ctx, testenv, diF, metav1.Now())).ToNot(HaveOccurred())

		testutils.ShouldReconcile(ctx, mockController, diReqS)
		testutils.ShouldReconcile(ctx, mockController, diReqS)
		testutils.ShouldReconcile(ctx, mockController, diReqF)
		testutils.ShouldReconcile(ctx, mockController, diReqF)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReqS.NamespacedName, diS))
		utils.ExpectNoError(testenv.Client.Get(ctx, diReqF.NamespacedName, diF))

		Expect(utils2.IsDeployItemPhase(diS, lsv1alpha1.DeployItemPhases.Succeeded)).To(BeTrue())
		Expect(utils2.IsDeployItemJobIDsIdentical(diS)).To(BeTrue())
		Expect(utils2.IsDeployItemPhase(diF, lsv1alpha1.DeployItemPhases.Failed)).To(BeTrue())
		Expect(utils2.IsDeployItemJobIDsIdentical(diF)).To(BeTrue())

		Expect(diS.Status.LastReconcileTime).NotTo(BeNil())
		Expect(diF.Status.LastReconcileTime).NotTo(BeNil())

		By("Set timed out LastReconcileTime timestamp")
		timedOut := metav1.Time{Time: time.Now().Add(-(testProgressingTimeoutDuration.Duration + (5 * time.Second)))}
		diS.Status.LastReconcileTime = &timedOut
		diS.Status.JobIDGenerationTime = &timedOut
		diF.Status.LastReconcileTime = &timedOut
		diF.Status.JobIDGenerationTime = &timedOut
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
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(testutils.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

		testutils.ShouldReconcile(ctx, mockController, diReq)
		testutils.ShouldReconcile(ctx, mockController, diReq)

		// verify state
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(utils2.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Progressing)).To(BeTrue())
		Expect(utils2.IsDeployItemJobIDsIdentical(di)).To(BeFalse())

		By("Set timed out LastReconcileTime timestamp (using default timeout duration)")
		timedOut := metav1.Time{Time: time.Now().Add(-(testProgressingTimeoutDuration.Duration + (5 * time.Second)))}
		di.Status.LastReconcileTime = &timedOut
		di.Status.JobIDGenerationTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		By("Verify that deploy item is not timed out")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Annotations).To(BeNil())

		By("Set timed out LastReconcileTime timestamp (deploy item specific timeout duration)")
		timedOut = metav1.Time{Time: time.Now().Add(-(di.Spec.Timeout.Duration + (5 * time.Second)))}
		di.Status.LastReconcileTime = &timedOut
		di.Status.JobIDGenerationTime = &timedOut
		utils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		By("Verify that deploy item is timed out")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(utils2.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Failed)).To(BeTrue())
		Expect(di.Status.LastError.Reason).To(Equal(lsv1alpha1.ProgressingTimeoutReason))
		Expect(di.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
		Expect(utils2.IsDeployItemJobIDsIdentical(di)).To(BeTrue())
	})
})
