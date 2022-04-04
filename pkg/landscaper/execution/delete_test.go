// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution_test

import (
	"context"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

	var (
		op *operation.Operation

		fakeExecutions  map[string]*lsv1alpha1.Execution
		fakeDeployItems map[string]*lsv1alpha1.DeployItem
		fakeClient      client.Client
	)

	BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		fakeExecutions = state.Executions
		fakeDeployItems = state.DeployItems
		op = operation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024))
	})

	It("should block deletion if deploy items still exist", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(1))
	})

	It("should delete the execution if no deploy items exist", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test1/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(0))
	})

	It("should delete all deploy items in reverse order", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(1))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-b", Namespace: "test3"}, item)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		item = &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-a", Namespace: "test3"}, item)
		Expect(err).ToNot(HaveOccurred())

		err = eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(1))

		item = &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-a", Namespace: "test3"}, item)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		err = eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(0))
	})

	It("should wait until a deploy item is deleted", func() {
		ctx := context.Background()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		deployItemB := fakeDeployItems["test3/di-b"]
		delTime := metav1.Now()
		deployItemB.DeletionTimestamp = &delTime
		controllerutil.AddFinalizer(deployItemB, lsv1alpha1.LandscaperFinalizer)
		Expect(eOp.Client().Update(ctx, deployItemB)).ToNot(HaveOccurred())

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(1))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-b", Namespace: "test3"}, item)
		Expect(err).ToNot(HaveOccurred())

		item = &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-a", Namespace: "test3"}, item)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should propagate an failure in a deploy item to the execution", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		deployItemB := fakeDeployItems["test3/di-b"]
		delTime := metav1.Now()
		deployItemB.DeletionTimestamp = &delTime
		deployItemB.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		controllerutil.AddFinalizer(deployItemB, lsv1alpha1.LandscaperFinalizer)
		Expect(eOp.Client().Update(ctx, deployItemB)).ToNot(HaveOccurred())

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(1))
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-b", Namespace: "test3"}, item)
		Expect(err).ToNot(HaveOccurred())

		item = &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-a", Namespace: "test3"}, item)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should propagate the delete-without-uninstall annotation to deploy items", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test6/exec-1"]

		eOp := execution.NewOperation(op, exec, false)
		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())

		deployItemA := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-a", Namespace: "test6"}, deployItemA)
		Expect(err).ToNot(HaveOccurred())
		Expect(lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(deployItemA.ObjectMeta)).To(BeTrue())

		deployItemB := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: "di-b", Namespace: "test6"}, deployItemB)
		Expect(err).ToNot(HaveOccurred())
		Expect(lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(deployItemB.ObjectMeta)).To(BeTrue())
	})
})
