// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution_test

import (
	"context"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Reconcile", func() {

	var (
		op operation.Interface

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
		op = operation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, nil)
	})

	It("should deploy the specified deploy items", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test1/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), item)
		Expect(err).ToNot(HaveOccurred())

		Expect(item.Spec.Type).To(Equal(lsv1alpha1.DeployItemType("landscaper.gardener.cloud/helm")))
		Expect(exec.Status.DeployItemReferences[0].Reference.ObservedGeneration).To(Equal(item.Generation))
	})

	It("should not deploy a deployitem when dependent ones haven't finished yet", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())

		// 2 because the dag starts with 2 parallel ones
		Expect(exec.Status.DeployItemReferences).To(HaveLen(2))

		// check that the last one is not referenced
		references := exec.Status.DeployItemReferences
		for _, reference := range references {
			Expect(reference.Name).ToNot(Equal("c"))
		}
	})

	It("should deploy the next deployitem when the previous one successfully finished", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		deployItemA := fakeDeployItems["test2/di-a"]
		deployItemA.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		Expect(fakeClient.Status().Update(ctx, deployItemA)).ToNot(HaveOccurred())

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(exec.Status.DeployItemReferences).To(HaveLen(2))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, exec.Status.DeployItemReferences[1].Reference.NamespacedName(), item)
		Expect(err).ToNot(HaveOccurred())

		Expect(item.Spec.Type).To(Equal(lsv1alpha1.DeployItemType("landscaper.gardener.cloud/helm")))
		Expect(exec.Status.DeployItemReferences[1].Reference.ObservedGeneration).To(Equal(exec.Generation))
	})

	It("should set the status of the execution to failed if a execution failed", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		deployItemA := fakeDeployItems["test2/di-a"]
		deployItemA.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		Expect(fakeClient.Status().Update(ctx, deployItemA)).ToNot(HaveOccurred())

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))
	})

	It("should not deploy new items if a execution failed", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec, false)

		deployItemA := fakeDeployItems["test2/di-a"]
		deployItemA.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		Expect(fakeClient.Status().Update(ctx, deployItemA)).ToNot(HaveOccurred())

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))
	})

})
