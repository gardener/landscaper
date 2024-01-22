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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	dictrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	utils2 "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/utils"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Deploy Item Controller Reconcile Test", func() {

	var (
		state                *envtest.State
		deployItemController reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error

		deployItemController, err = dictrl.NewController(testenv.Client, testenv.Client, logging.Discard(), api.LandscaperScheme,
			&testPickupTimeoutDuration, 1000)
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

	It("should detect pickup timeouts", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Get test deploy items")
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
		Expect(di.Status.LastError).ToNot(BeNil())
		Expect(di.Status.LastError.Message).ToNot(ContainSubstring("Target"))
	})

	It("should detect if the reason for a pickup timeout is a missing target", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, testdataDir)
		Expect(err).ToNot(HaveOccurred())

		By("Get test deploy items")
		di := &lsv1alpha1.DeployItem{}
		diReq := testutils.Request("mock-di-prog", state.Namespace)
		// do not reconcile with mock deployer

		By("Set timed out reconcile timestamp annotation")
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		// add reference to non-existing Target
		di.Spec.Target = &lsv1alpha1.ObjectReference{
			Name: "null",
		}
		utils.ExpectNoError(testenv.Client.Update(ctx, di))
		timedOut := metav1.Time{Time: time.Now().Add(-(testPickupTimeoutDuration.Duration + (5 * time.Second)))}

		Expect(testutils.UpdateJobIdForDeployItem(ctx, testenv, di, timedOut)).ToNot(HaveOccurred())

		By("Verify that timed out deploy items are in 'Failed' phase")
		testutils.ShouldReconcile(ctx, deployItemController, diReq)
		utils.ExpectNoError(testenv.Client.Get(ctx, diReq.NamespacedName, di))
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed))
		Expect(utils2.IsDeployItemJobIDsIdentical(di)).To(BeTrue())
		Expect(di.Status.LastError).ToNot(BeNil())
		Expect(di.Status.LastError.Message).To(ContainSubstring("Target"))
	})

})
