// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution_test

import (
	"context"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ = Describe("Reconcile", func() {

	var (
		ctrl  reconcile.Reconciler
		state *envtest.State
	)
	BeforeEach(func() {
		var err error
		ctrl, err = execution.NewController(logr.Discard(), testenv.Client, api.Scheme, record.NewFakeRecorder(1024))
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

		if utils.IsNewReconcile() {
			Expect(state.Create(ctx, exec)).To(Succeed())
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecPhaseProgressing))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(exec.Status.JobID))

			// expect a deploy item
			items := &lsv1alpha1.DeployItemList{}
			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
			Expect(items.Items).To(HaveLen(1))

			// set item to failed state
			di := &items.Items[0]
			di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
			di.Status.JobID = exec.Status.JobID
			di.Status.JobIDFinished = exec.Status.JobID
			testutils.ExpectNoError(state.Client.Status().Update(ctx, di))

			// reconcile execution and check that it is failed
			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecPhaseFailed))
			Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))

			// delete execution
			testutils.ExpectNoError(testenv.Client.Delete(ctx, exec))

			// reconcile execution and check that objects are gone
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))).To(BeTrue(), "expect the deploy item to be deleted")
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))).To(BeTrue(), "expect the execution to be deleted")
		} else {
			testutils.ExpectNoError(state.Create(ctx, exec))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			// expect a deploy item
			items := &lsv1alpha1.DeployItemList{}
			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
			Expect(items.Items).To(HaveLen(1))
			di := &items.Items[0]
			//set item to failed state
			di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed

			// then reconcile the execution and delete it
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			testutils.ExpectNoError(testenv.Client.Delete(ctx, exec))
			// reconcile 2 times so that the deployitem is deleted on the first
			// and on the execution on the second reconcile
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))).To(BeTrue(), "expect the deploy item to be deleted")
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))).To(BeTrue(), "expect the execution to be deleted")
		}
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
			if utils.IsNewReconcile() {
				_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			} else {
				testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			}

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
			if utils.IsNewReconcile() {
				_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			} else {
				testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			}

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
		if utils.IsNewReconcile() {
			Expect(state.Create(ctx, exec)).To(Succeed())
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())

			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecPhaseProgressing))
			Expect(exec.Status.JobIDFinished).NotTo(Equal(exec.Status.JobID))

			// expect a deploy item
			items := &lsv1alpha1.DeployItemList{}
			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
			Expect(items.Items).To(HaveLen(1))
			di := &items.Items[0]

			// set item to failed state
			di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
			di.Status.JobID = exec.Status.JobID
			di.Status.JobIDFinished = exec.Status.JobID
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

			// then reconcile the execution and expect the execution to be Failed
			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecPhaseFailed))
			Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))

			// set deploy item phase to Succeeded and check again
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())

			di.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
			di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseSucceeded
			di.Status.JobID = exec.Status.JobID
			di.Status.JobIDFinished = exec.Status.JobID
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecPhaseSucceeded))
			Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))

			// verify that from Succeeded to Failed also works
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(testutils.UpdateJobIdForExecution(ctx, testenv, exec)).To(Succeed())

			di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
			di.Status.JobID = exec.Status.JobID
			di.Status.JobIDFinished = exec.Status.JobID
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecPhaseFailed))
			Expect(exec.Status.JobIDFinished).To(Equal(exec.Status.JobID))
		} else {
			Expect(state.Create(ctx, exec)).To(Succeed())
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))

			// expect a deploy item
			items := &lsv1alpha1.DeployItemList{}
			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
			Expect(items.Items).To(HaveLen(1))
			di := &items.Items[0]

			// set item to failed state
			di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			di.Status.ObservedGeneration = di.Generation
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))

			// then reconcile the execution and expect the execution to be Failed
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))

			// set deployitem phase to Succeeded and check again
			di.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// verify that from Succeeded to Failed also works
			di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))
		}
	})

})
