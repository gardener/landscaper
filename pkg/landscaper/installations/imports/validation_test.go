// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Validation", func() {

	var (
		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		fakeCompRepo      ctf.ComponentResolver
	)

	BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespace(fakeClient)
		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(fakeCompRepo),
		}
	})

	It("should successfully validate when the import of a component is defined by its parent", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewTestValidator(op, inInstRoot)
		Expect(val.ImportsSatisfied(ctx, inInstA)).To(Succeed())
	})

	It("should successfully validate when the data import of a component is defined by a sibling and all sibling dependencies are completed", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewTestValidator(op, inInstRoot)
		Expect(val.ImportsSatisfied(ctx, inInstB)).To(Succeed())
	})

	It("should successfully validate when the target import of a component is defined by a sibling and all sibling dependencies are completed", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstE, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/e"])
		Expect(err).ToNot(HaveOccurred())
		inInstE.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/f"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstF
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewTestValidator(op, inInstRoot)
		Expect(val.ImportsSatisfied(ctx, inInstF)).To(Succeed())
	})

	It("should reject the validation when the parent component is not progressing", func() {
		ctx := context.Background()
		inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())
		inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseInit
		Expect(fakeClient.Status().Update(ctx, inInstRoot.Info))

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewTestValidator(op, inInstRoot)
		err = val.ImportsSatisfied(ctx, inInstA)
		Expect(err).To(HaveOccurred())
		Expect(installations.IsImportNotSatisfiedError(err)).To(BeTrue())
	})

	It("should reject when a direct sibling dependency is still running", func() {
		ctx := context.Background()

		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewTestValidator(op, inInstRoot)
		err = val.ImportsSatisfied(ctx, inInstB)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should reject when a dependent sibling has not finished yet", func() {
		ctx := context.Background()
		inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
		Expect(fakeClient.Update(ctx, inInstA.Info)).To(Succeed())

		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		inInstB.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
		Expect(err).ToNot(HaveOccurred())
		inInstC.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstD

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewTestValidator(op, inInstRoot)
		err = val.ImportsSatisfied(ctx, inInstD)
		Expect(err).To(HaveOccurred())
		Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
	})

	Context("CheckDependentInstallations", func() {
		It("should reject when a dependent sibling of my parent that has not finished yet", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test3/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test3/root"])
			Expect(err).ToNot(HaveOccurred())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			val := imports.NewTestValidator(op, inInstRoot)
			ok, err := val.CheckDependentInstallations(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Context("Targets", func() {
		It("should forbid if a target import from a manually added target is not present", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			target := &lsv1alpha1.Target{}
			target.Name = lsv1alpha1helper.GenerateDataObjectName("", "ext.a")
			target.Namespace = "test4"
			Expect(fakeClient.Delete(ctx, target))

			val := imports.NewTestValidator(op, nil)
			err = val.ImportsSatisfied(ctx, inInstRoot)
			Expect(err).To(HaveOccurred())
		})

		It("should forbid if a import from a parent import is not present", func() {
			ctx := context.Background()
			inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/f"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstF
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			target := &lsv1alpha1.Target{}
			target.Name = lsv1alpha1helper.GenerateDataObjectName(op.Context().Name, "root.a")
			target.Namespace = "test4"
			Expect(fakeClient.Delete(ctx, target))

			val := imports.NewTestValidator(op, nil)
			err = val.ImportsSatisfied(ctx, inInstF)
			Expect(err).To(HaveOccurred())
		})
	})

})
