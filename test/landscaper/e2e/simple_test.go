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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	"github.com/gardener/landscaper/pkg/kubernetes"
	execctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	instctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Simple", func() {

	var (
		state        *envtest.State
		fakeRegistry blueprintsregistry.Registry

		execActuator, instActuator, mockActuator reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error
		fakeRegistry, err = blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, []string{filepath.Join(projectRoot, "examples", "01-simple", "definitions")})
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

	It("Should successfully reconcile SimpleTest", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, filepath.Join(projectRoot, "examples", "01-simple", "cluster"))
		Expect(err).ToNot(HaveOccurred())

		// first the installation controller should run and set the finalizer
		// afterwards it should again reconcile and deploy the execution
		instReq := request("root-1", state.Namespace)
		testutils.ShouldReconcile(instActuator, instReq)
		testutils.ShouldReconcile(instActuator, instReq)

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
		_, err = instActuator.Reconcile(request("root-1", state.Namespace))
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))
		Expect(inst.Status.ExportReference).ToNot(BeNil())

		By("delete resource")
		Expect(testenv.Client.Delete(ctx, inst)).ToNot(HaveOccurred())

		// the installation controller should propagate the deletion to its subcharts
		_, err = instActuator.Reconcile(instReq)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("waiting for deletion"))

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")

		// the execution controller should propagate the deletion to its deploy item
		_, err = execActuator.Reconcile(execReq)
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).ToNot(HaveOccurred())
		Expect(di.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")

		_, err = mockActuator.Reconcile(diReq)
		Expect(err).ToNot(HaveOccurred())
		err = testenv.Client.Get(ctx, diReq.NamespacedName, di)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "deploy item should be deleted")

		// execution controller should remove the finalizer
		testutils.ShouldReconcile(execActuator, execReq)
		err = testenv.Client.Get(ctx, execReq.NamespacedName, exec)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "execution should be deleted")

		// installation controller should remove its own finalizer
		testutils.ShouldReconcile(instActuator, instReq)
		err = testenv.Client.Get(ctx, instReq.NamespacedName, inst)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "installation should be deleted")
	})
})

func request(name, namespace string) reconcile.Request {
	req := reconcile.Request{}
	req.Name = name
	req.Namespace = namespace
	return req
}
