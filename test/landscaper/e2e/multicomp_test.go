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

package e2e_test

import (
	"context"
	"path/filepath"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	"github.com/gardener/landscaper/pkg/kubernetes"
	execctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	instctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Multi Component Test", func() {

	var (
		state        *envtest.State
		fakeRegistry registry.Registry

		execActuator, instActuator, mockActuator reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error
		fakeRegistry, err = registry.NewLocalRegistry(testing.NullLogger{}, []string{filepath.Join(projectRoot, "examples", "02-multi-comp", "definitions")})
		Expect(err).ToNot(HaveOccurred())

		instActuator, err = instctlr.NewActuator(fakeRegistry)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.ClientInto(testenv.Client, instActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.SchemeInto(kubernetes.LandscaperScheme, instActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.LoggerInto(testing.NullLogger{}, instActuator)
		Expect(err).ToNot(HaveOccurred())

		execActuator, err = execctlr.NewActuator(fakeRegistry)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.ClientInto(testenv.Client, execActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.SchemeInto(kubernetes.LandscaperScheme, execActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.LoggerInto(testing.NullLogger{}, execActuator)
		Expect(err).ToNot(HaveOccurred())

		mockActuator, err = mockctlr.NewActuator()
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.ClientInto(testenv.Client, mockActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.SchemeInto(kubernetes.LandscaperScheme, mockActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.LoggerInto(testing.NullLogger{}, mockActuator)
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

	It("Should successfully reconcile MultiCompTest", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, filepath.Join(projectRoot, "examples", "02-multi-comp", "cluster"))
		Expect(err).ToNot(HaveOccurred())

		var (
			instReq  = request("root-1", state.Namespace)
			inst2Req = request("root-2", state.Namespace)
		)

		testutils.ShouldReconcile(instActuator, inst2Req)
		testutils.ShouldNotReconcile(instActuator, inst2Req, "should not reconcile the second component as there are dependencies on exports of root-1")

		// first the installation controller should run and set the finalizer
		// afterwards it should again reconcile and deploy the execution
		testutils.ShouldReconcile(instActuator, instReq)
		testutils.ShouldReconcile(instActuator, instReq)
		testutils.ShouldNotReconcile(instActuator, inst2Req, "should not reconcile the second component as there are still dependencies on root-1")

		inst := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseProgressing))
		Expect(inst.Status.ExecutionReference).ToNot(BeNil())

		execReq := request(inst.Status.ExecutionReference.Name, inst.Status.ExecutionReference.Namespace)
		exec := &lsv1alpha1.Execution{}
		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())

		// after the execution was created by the installation, we need to run the execution controller
		// on first reconcile it should add the finalizer
		// and int he second reconcile it should create the deploy item
		testutils.ShouldReconcile(execActuator, execReq)
		testutils.ShouldReconcile(execActuator, execReq)
		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))

		diList := &lsv1alpha1.DeployItemList{}
		Expect(testenv.Client.List(ctx, diList)).ToNot(HaveOccurred())
		Expect(diList.Items).To(HaveLen(1))

		diReq := request(exec.Status.DeployItemReferences[0].Reference.Name, exec.Status.DeployItemReferences[0].Reference.Namespace)
		di := &lsv1alpha1.DeployItem{}
		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).ToNot(HaveOccurred())
		testutils.ShouldReconcile(mockActuator, diReq)
		testutils.ShouldReconcile(mockActuator, diReq)

		// as the deploy item is now successfully reconciled, we have to trigger the execution
		// and check if the states are correctly propagated
		_, err = execActuator.Reconcile(execReq)
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(exec.Status.ExportReference).ToNot(BeNil())

		// as the execution is now successfully reconciled, we have to trigger the installation
		// and check if the state is propagated
		testutils.ShouldReconcile(instActuator, instReq)
		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))
		Expect(inst.Status.ExportReference).ToNot(BeNil())

		By("as the first component is now successfully reconciled, we should be able to reconcile the second component")
		testutils.ShouldReconcile(instActuator, inst2Req)
		Expect(testenv.Client.Get(ctx, inst2Req.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseProgressing))
		Expect(inst.Status.ExecutionReference).ToNot(BeNil())

		By("reconcile the execution of root-2 2 times and create a deploy item")
		exec2Req := request(inst.Status.ExecutionReference.Name, inst.Status.ExecutionReference.Namespace)
		testutils.ShouldReconcile(execActuator, exec2Req)
		testutils.ShouldReconcile(execActuator, exec2Req)
		Expect(testenv.Client.Get(ctx, exec2Req.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))

		Expect(testenv.Client.List(ctx, diList)).ToNot(HaveOccurred())
		Expect(diList.Items).To(HaveLen(2), "there should be one deployitem from root-1 and one from root-2")

		By("reconcile the deploy item")
		di2Req := request(exec.Status.DeployItemReferences[0].Reference.Name, exec.Status.DeployItemReferences[0].Reference.Namespace)
		testutils.ShouldReconcile(mockActuator, di2Req)
		testutils.ShouldReconcile(mockActuator, di2Req)

		By("as the deploy item is now successfully reconciled, we have to trigger the execution and check if the states are correctly propagated")
		testutils.ShouldReconcile(execActuator, exec2Req)
		Expect(testenv.Client.Get(ctx, exec2Req.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(exec.Status.ExportReference).ToNot(BeNil())

		By("should be unable to delete root-1")
		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(testenv.Client.Delete(ctx, inst)).ToNot(HaveOccurred())
		testutils.ShouldNotReconcile(instActuator, instReq, "the component should not be deleted as there are still components that import its exports")

		By("should delete root-2")
		testutils.DeleteInstallation(ctx, testenv.Client, execActuator, instActuator, mockActuator, inst2Req)

		By("should delete root-1")
		testutils.DeleteInstallation(ctx, testenv.Client, execActuator, instActuator, mockActuator, instReq)
	})
})
