// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package execution_test

import (
	"context"

	"github.com/go-logr/logr/testing"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = g.Describe("Reconcile", func() {

	var (
		op operation.Interface

		fakeExecutions  map[string]*lsv1alpha1.Execution
		fakeDeployItems map[string]*lsv1alpha1.DeployItem
		fakeClient      client.Client
	)

	g.BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		fakeExecutions = state.Executions
		fakeDeployItems = state.DeployItems
		op = operation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, nil, nil)
	})

	g.It("should deploy the specified deploy items", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test1/exec-1"]
		eOp := execution.NewOperation(op, exec)

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), item)
		Expect(err).ToNot(HaveOccurred())

		Expect(item.Spec.ImportReference).To(Equal(exec.Spec.ImportReference))
		Expect(item.Spec.Type).To(Equal(lsv1alpha1.ExecutionType("Helm")))
		Expect(exec.Status.DeployItemReferences[0].Reference.ObservedGeneration).To(Equal(item.Generation))
	})

	g.It("should not deploy the next deployitem when the previous one has not finished yet", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec)

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))
	})

	g.It("should deploy the next deployitem when the previous one successfully finished", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec)

		deployItemA := fakeDeployItems["test2/di-a"]
		deployItemA.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		Expect(fakeClient.Status().Update(ctx, deployItemA)).ToNot(HaveOccurred())

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(exec.Status.DeployItemReferences).To(HaveLen(2))

		item := &lsv1alpha1.DeployItem{}
		err = fakeClient.Get(ctx, exec.Status.DeployItemReferences[1].Reference.NamespacedName(), item)
		Expect(err).ToNot(HaveOccurred())

		Expect(item.Spec.ImportReference).To(Equal(exec.Spec.ImportReference))
		Expect(item.Spec.Type).To(Equal(lsv1alpha1.ExecutionType("Container")))
		Expect(exec.Status.DeployItemReferences[1].Reference.ObservedGeneration).To(Equal(exec.Generation))
	})

	g.It("should set the status of the execution to failed if a execution failed", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec)

		deployItemA := fakeDeployItems["test2/di-a"]
		deployItemA.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		Expect(fakeClient.Status().Update(ctx, deployItemA)).ToNot(HaveOccurred())

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseFailed))
	})

	g.It("should not deploy new items if a execution failed", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test2/exec-1"]
		eOp := execution.NewOperation(op, exec)

		deployItemA := fakeDeployItems["test2/di-a"]
		deployItemA.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		Expect(fakeClient.Status().Update(ctx, deployItemA)).ToNot(HaveOccurred())

		err := eOp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))
	})

})
