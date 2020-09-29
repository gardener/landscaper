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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

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
		op = operation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, nil, nil)
	})

	It("should block deletion if deploy items still exist", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec)

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(1))
	})

	It("should delete the execution if no deploy items exist", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test1/exec-1"]
		eOp := execution.NewOperation(op, exec)

		err := eOp.Delete(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(exec.Finalizers).To(HaveLen(0))
	})

	It("should delete all deploy items in reverse order", func() {
		ctx := context.Background()
		defer ctx.Done()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec)

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
		defer ctx.Done()
		exec := fakeExecutions["test3/exec-1"]
		eOp := execution.NewOperation(op, exec)

		deployItemB := fakeDeployItems["test3/di-b"]
		delTime := metav1.Now()
		deployItemB.DeletionTimestamp = &delTime
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
		eOp := execution.NewOperation(op, exec)

		deployItemB := fakeDeployItems["test3/di-b"]
		delTime := metav1.Now()
		deployItemB.DeletionTimestamp = &delTime
		deployItemB.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
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

})
