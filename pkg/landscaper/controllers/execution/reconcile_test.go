// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution_test

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	"github.com/gardener/landscaper/test/utils/envtest"
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

//	It("should correctly reconcile a deleted execution when it was in failed state", func() {
//		ctx := context.Background()
//		// first deploy reconcile a simple execution with one deploy item
//		exec := &lsv1alpha1.Execution{}
//		exec.GenerateName = "test-"
//		exec.Namespace = state.Namespace
//		exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
//			{
//				Name: "def",
//				Type: "test-type",
//				Configuration: &runtime.RawExtension{
//					Raw: []byte(`
//{
//  "apiVersion": "sometest",
//  "kind": "somekind"
//}
//`),
//				},
//			},
//		}
//		testutils.ExpectNoError(state.Create(ctx, exec))
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//
//		// expect a deploy item
//		items := &lsv1alpha1.DeployItemList{}
//		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
//		Expect(items.Items).To(HaveLen(1))
//		di := &items.Items[0]
//		//set item to failed state
//		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
//
//		// then reconcile the execution and delete it
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//		testutils.ExpectNoError(testenv.Client.Delete(ctx, exec))
//		// reconcile 2 times so that the deployitem is deleted on the first
//		// and on the execution on the second reconcile
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//
//		Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))).To(BeTrue(), "expect the deploy item to be deleted")
//		Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))).To(BeTrue(), "expect the execution to be deleted")
//	})
//
//	Context("Context", func() {
//		It("should pass the context to the deploy item", func() {
//			ctx := context.Background()
//			// first deploy reconcile a simple execution with one deploy item
//			exec := &lsv1alpha1.Execution{}
//			exec.GenerateName = "test-"
//			exec.Namespace = state.Namespace
//			exec.Spec.Context = "test"
//			exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
//				{
//					Name: "def",
//					Type: "test-type",
//					Configuration: &runtime.RawExtension{
//						Raw: []byte(`
//{
//  "apiVersion": "sometest",
//  "kind": "somekind"
//}
//`),
//					},
//				},
//			}
//			testutils.ExpectNoError(state.Create(ctx, exec))
//			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//
//			// expect a deploy item
//			items := &lsv1alpha1.DeployItemList{}
//			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
//			Expect(items.Items).To(HaveLen(1))
//			di := &items.Items[0]
//			Expect(di.Spec.Context).To(Equal("test"))
//		})
//
//		It("should default the context of the deploy item", func() {
//			ctx := context.Background()
//			// first deploy reconcile a simple execution with one deploy item
//			exec := &lsv1alpha1.Execution{}
//			exec.GenerateName = "test-"
//			exec.Namespace = state.Namespace
//			exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
//				{
//					Name: "def",
//					Type: "test-type",
//					Configuration: &runtime.RawExtension{
//						Raw: []byte(`
//{
//  "apiVersion": "sometest",
//  "kind": "somekind"
//}
//`),
//					},
//				},
//			}
//			testutils.ExpectNoError(state.Create(ctx, exec))
//			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//
//			// expect a deploy item
//			items := &lsv1alpha1.DeployItemList{}
//			testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
//			Expect(items.Items).To(HaveLen(1))
//			di := &items.Items[0]
//			Expect(di.Spec.Context).To(Equal("default"))
//		})
//	})
//
//	It("should adapt the status of the execution if a deploy item changes from Failed to Succeeded and vice versa", func() {
//		ctx := context.Background()
//		// first deploy reconcile a simple execution with one deploy item
//		exec := &lsv1alpha1.Execution{}
//		exec.GenerateName = "test-"
//		exec.Namespace = state.Namespace
//		exec.Spec.DeployItems = []lsv1alpha1.DeployItemTemplate{
//			{
//				Name: "def",
//				Type: "test-type",
//				Configuration: &runtime.RawExtension{
//					Raw: []byte(`
//{
//  "apiVersion": "sometest",
//  "kind": "somekind"
//}
//`),
//				},
//			},
//		}
//		testutils.ExpectNoError(state.Create(ctx, exec))
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//
//		// expect a deploy item
//		items := &lsv1alpha1.DeployItemList{}
//		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
//		Expect(items.Items).To(HaveLen(1))
//		di := &items.Items[0]
//		//set item to failed state
//		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
//		di.Status.ObservedGeneration = di.Generation
//		testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))
//
//		// then reconcile the execution and expect the execution to be Failed
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
//		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))
//
//		// set deployitem phase to Succeeded and check again
//		di.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
//		testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
//		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
//
//		// verify that from Succeeded to Failed also works
//		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
//		testutils.ExpectNoError(testenv.Client.Status().Update(ctx, di))
//		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(exec))
//		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
//		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))
//	})

})
