// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution_test

import (
	"context"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Reconcile", func() {

	var (
		ctrl  reconcile.Reconciler
		state *envtest.State
	)
	BeforeEach(func() {
		var err error
		ctrl, err = execution.NewController(logging.Discard(), testenv.Client, testenv.Client, api.Scheme,
			record.NewFakeRecorder(1024), 1000, false, "exec-test-"+testutils.GetNextCounter())
		Expect(err).ToNot(HaveOccurred())
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	It("should correctly reconcile a deleted execution when it was in failed state", func() {
		ctx := context.Background()
		// first deploy reconcile a simple execution with one deploy item
		exec := &lsv1alpha1.Execution{}
		exec.GenerateName = "test-"
		exec.Namespace = state.Namespace
		exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
			{
				Name: "def",
				Type: "test-type",
				Configuration: &runtime.RawExtension{
					Raw: []byte(`
{
  "apiVersion": "sometest",
  "kind": "somekind"
}
`),
				},
			},
		}

		Expect(state.Create(ctx, exec)).To(Succeed())
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))
		Expect(exec.Status.JobIDFinished).NotTo(Equal(exec.Status.JobID))

		// expect a deploy item
		items := &lsv1alpha1.DeployItemList{}
		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
		Expect(items.Items).To(HaveLen(1))

		// set item to failed state
		di := &items.Items[0]
		di.Status.Phase = lsv1alpha1.DeployItemPhases.Failed
		di.Status.SetJobID(exec.Status.JobID)
		di.Status.JobIDFinished = exec.Status.JobID
		testutils.ExpectNoError(state.Client.Status().Update(ctx, di))

		// reconcile execution and check that it is failed
		testutils.ShouldReconcileButRetry(ctx, ctrl, testutils.RequestFromObject(exec))
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Failed))
		Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))

		// delete execution
		testutils.ExpectNoError(testenv.Client.Delete(ctx, exec))

		// reconcile execution and check that objects are gone
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
		Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))).To(BeTrue(), "expect the deploy item to be deleted")
		Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))).To(BeTrue(), "expect the execution to be deleted")
	})

	Context("Context", func() {
		It("should pass the context to the deploy item", func() {
			ctx := context.Background()
			// first deploy reconcile a simple execution with one deploy item
			exec := &lsv1alpha1.Execution{}
			exec.GenerateName = "test-"
			exec.Namespace = state.Namespace
			exec.Spec.Context = "test"
			exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
				{
					Name: "def",
					Type: "test-type",
					Configuration: &runtime.RawExtension{
						Raw: []byte(`
{
  "apiVersion": "sometest",
  "kind": "somekind"
}
`),
					},
				},
			}
			Expect(state.Create(ctx, exec)).To(Succeed())
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			// expect a deploy item
			items := &lsv1alpha1.DeployItemList{}
			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
			Expect(items.Items).To(HaveLen(1))
			di := &items.Items[0]
			Expect(di.Spec.Context).To(Equal("test"))
		})

		It("should default the context of the deploy item", func() {
			ctx := context.Background()
			// first deploy reconcile a simple execution with one deploy item
			exec := &lsv1alpha1.Execution{}
			exec.GenerateName = "test-"
			exec.Namespace = state.Namespace
			exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
				{
					Name: "def",
					Type: "test-type",
					Configuration: &runtime.RawExtension{
						Raw: []byte(`
{
  "apiVersion": "sometest",
  "kind": "somekind"
}
`),
					},
				},
			}
			Expect(state.Create(ctx, exec)).To(Succeed())
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			// expect a deploy item
			items := &lsv1alpha1.DeployItemList{}
			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
			Expect(items.Items).To(HaveLen(1))
			di := &items.Items[0]
			Expect(di.Spec.Context).To(Equal("default"))
		})
	})

	It("should adapt the status of the execution if a deploy item changes from Failed to Succeeded and vice versa", func() {
		ctx := context.Background()
		// first deploy reconcile a simple execution with one deploy item
		exec := &lsv1alpha1.Execution{}
		exec.GenerateName = "test-"
		exec.Namespace = state.Namespace
		exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
			{
				Name: "def",
				Type: "test-type",
				Configuration: &runtime.RawExtension{
					Raw: []byte(`
{
  "apiVersion": "sometest",
  "kind": "somekind"
}
`),
				},
			},
		}
		Expect(state.Create(ctx, exec)).To(Succeed())
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))
		Expect(exec.Status.JobIDFinished).NotTo(Equal(exec.Status.JobID))

		// expect a deploy item
		items := &lsv1alpha1.DeployItemList{}
		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
		Expect(items.Items).To(HaveLen(1))
		di := &items.Items[0]

		// set item to failed state
		di.Status.Phase = lsv1alpha1.DeployItemPhases.Failed
		di.Status.SetJobID(exec.Status.JobID)
		di.Status.JobIDFinished = exec.Status.JobID
		testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		// then reconcile the execution and expect the execution to be Failed
		testutils.ShouldReconcileButRetry(ctx, ctrl, testutils.RequestFromObject(exec))
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Failed))
		Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))

		// set deploy item phase to Succeeded and check again
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())

		di.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
		di.Status.SetJobID(exec.Status.JobID)
		di.Status.JobIDFinished = exec.Status.JobID
		testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Succeeded))
		Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))

		// verify that from Succeeded to Failed also works
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())

		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
		di.Status.Phase = lsv1alpha1.DeployItemPhases.Failed
		di.Status.SetJobID(exec.Status.JobID)
		di.Status.JobIDFinished = exec.Status.JobID
		testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		testutils.ShouldReconcileButRetry(ctx, ctrl, testutils.RequestFromObject(exec))
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Failed))
		Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))
	})

	Context("Cleanup deploy items", func() {

		It("should cleanup orphaned deploy items", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/test1")
			testutils.ExpectNoError(err)
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			// Remove two deploy items di-b and di-c from the execution spec
			exec := state.Executions[state.Namespace+"/exec-1"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Spec.DeployItems).To(HaveLen(3))
			exec.Spec.DeployItems = exec.Spec.DeployItems[:1]
			Expect(testenv.Client.Update(ctx, exec)).To(Succeed())

			// Reconcile the execution with two orphaned deploy items di-b and di-c
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			// Check execution state
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			deployItems, err := read_write_layer.ListManagedDeployItems(ctx, testenv.Client, client.ObjectKeyFromObject(exec), read_write_layer.R000000)
			Expect(err).To(BeNil())
			Expect(deployItems.Items).To(HaveLen(3))
			Expect(exec.Status.ExecutionGenerations).To(HaveLen(1))

			// Check that the first deploy item di-a is not deleted
			di := state.DeployItems[state.Namespace+"/di-a"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(di.DeletionTimestamp.IsZero()).To(BeTrue())

			// Check that the deletion of the orphaned deploy items di-b and di-c was triggered
			di = state.DeployItems[state.Namespace+"/di-b"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(di.DeletionTimestamp.IsZero()).To(BeFalse())
			Expect(di.Status.GetJobID()).To(Equal(exec.Status.JobID))

			di = state.DeployItems[state.Namespace+"/di-c"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(di.DeletionTimestamp.IsZero()).To(BeFalse())
			Expect(di.Status.GetJobID()).To(Equal(exec.Status.JobID))
		})
	})

	Context("Dependencies", func() {

		It("should trigger deploy items in the correct order", func() {
			ctx := context.Background()

			// We consider three deploy items a, b, c. Deploy item c depends on a and b.
			var err error
			state, err = testenv.InitResources(ctx, "./testdata/test2")
			testutils.ExpectNoError(err)
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			// Set a new jobID and reconcile.
			// Afterwards, deploy items a and b should be triggered, i.e. they should have the new jobID.
			// Deploy item c should not be triggered, as it depends on the other two.
			// The execution should be Progressing
			exec := state.Executions[state.Namespace+"/exec-2"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			currentJobID := exec.Status.JobID
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))

			di1 := state.DeployItems[state.Namespace+"/di-a"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di1), di1)).To(Succeed())
			Expect(di1.Status.JobID).To(Equal(currentJobID))
			Expect(di1.Status.JobIDFinished).NotTo(Equal(currentJobID))

			di2 := state.DeployItems[state.Namespace+"/di-b"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di2), di2)).To(Succeed())
			Expect(di2.Status.JobID).To(Equal(currentJobID))
			Expect(di2.Status.JobIDFinished).NotTo(Equal(currentJobID))

			di3 := state.DeployItems[state.Namespace+"/di-c"]
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).NotTo(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).NotTo(Equal(currentJobID))

			// Update di-a to Succeeded and di-b to Progressing, then reconcile the execution.
			// Afterwards, deploy item c should still not be triggered and the execution should still be Progressing.
			di1.Status.JobIDFinished = currentJobID
			di1.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
			Expect(state.Client.Status().Update(ctx, di1)).To(Succeed())

			di2.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing
			Expect(state.Client.Status().Update(ctx, di1)).To(Succeed())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).NotTo(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).NotTo(Equal(currentJobID))

			// Update di-b to Succeeded, then reconcile the execution.
			// Afterwards, deploy item c should be triggered, because its predecessors are finished.
			// The execution should still be Progressing.
			di2.Status.JobIDFinished = currentJobID
			di2.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
			Expect(state.Client.Status().Update(ctx, di2)).To(Succeed())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).To(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).NotTo(Equal(currentJobID))

			// Update di-c to Succeeded, then reconcile the execution.
			// Afterwards, the execution should be finished.
			di3.Status.JobIDFinished = currentJobID
			di3.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
			Expect(state.Client.Status().Update(ctx, di3)).To(Succeed())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).To(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Succeeded))

			// Change the dependencies of the deploy items: first a, then b, then c.
			// Then set a new jobID and reconcile.
			for i := range exec.Spec.DeployItems {
				di := &exec.Spec.DeployItems[i]
				switch di.Name {
				case "a":
					di.DependsOn = nil
				case "b":
					di.DependsOn = []string{"a"}
				case "c":
					di.DependsOn = []string{"b"}
				}
			}
			Expect(state.Update(ctx, exec)).NotTo(HaveOccurred())
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())

			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			currentJobID = exec.Status.JobID
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di1), di1)).To(Succeed())
			Expect(di1.Status.JobID).To(Equal(currentJobID))
			Expect(di1.Status.JobIDFinished).NotTo(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di2), di2)).To(Succeed())
			Expect(di2.Status.JobID).NotTo(Equal(currentJobID))
			Expect(di2.Status.JobIDFinished).NotTo(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).NotTo(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).NotTo(Equal(currentJobID))

			// Update di-a to Succeeded, then reconcile the execution.
			// Afterwards, deploy item b should be triggered.
			di1.Status.JobIDFinished = currentJobID
			di1.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
			Expect(state.Client.Status().Update(ctx, di1)).To(Succeed())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di1), di1)).To(Succeed())
			Expect(di1.Status.JobID).To(Equal(currentJobID))
			Expect(di1.Status.JobIDFinished).To(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di2), di2)).To(Succeed())
			Expect(di2.Status.JobID).To(Equal(currentJobID))
			Expect(di2.Status.JobIDFinished).NotTo(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).NotTo(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).NotTo(Equal(currentJobID))

			// Update di-b to Succeeded, then reconcile the execution.
			// Afterwards, deploy item c should be triggered.
			di2.Status.JobIDFinished = currentJobID
			di2.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
			Expect(state.Client.Status().Update(ctx, di2)).To(Succeed())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di1), di1)).To(Succeed())
			Expect(di1.Status.JobID).To(Equal(currentJobID))
			Expect(di1.Status.JobIDFinished).To(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di2), di2)).To(Succeed())
			Expect(di2.Status.JobID).To(Equal(currentJobID))
			Expect(di2.Status.JobIDFinished).To(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).To(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).NotTo(Equal(currentJobID))

			// Update di-c to Succeeded, then reconcile the execution.
			// Afterwards, the execution should be succeeded
			di3.Status.JobIDFinished = currentJobID
			di3.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
			Expect(state.Client.Status().Update(ctx, di3)).To(Succeed())

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.JobID).To(Equal(currentJobID))
			Expect(exec.Status.JobIDFinished).To(Equal(currentJobID))
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Succeeded))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di1), di1)).To(Succeed())
			Expect(di1.Status.JobID).To(Equal(currentJobID))
			Expect(di1.Status.JobIDFinished).To(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di2), di2)).To(Succeed())
			Expect(di2.Status.JobID).To(Equal(currentJobID))
			Expect(di2.Status.JobIDFinished).To(Equal(currentJobID))

			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di3), di3)).To(Succeed())
			Expect(di3.Status.JobID).To(Equal(currentJobID))
			Expect(di3.Status.JobIDFinished).To(Equal(currentJobID))
		})
	})
})
