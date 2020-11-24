// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("OutdatedImports", func() {

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

		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient(testing.NullLogger{}, "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Interface: lsoperation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeCompRepo),
		}
	})

	It("should return that imports are outdated if a import from the parent is outdated", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		op.Context().Parent = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		outdated, err := val.OutdatedImports(ctx, inInstA)
		Expect(err).To(Succeed())
		Expect(outdated).To(BeTrue())
	})

	It("should return that imports are outdated if a import from another component is outdated", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		instA := fakeInstallations["test1/a"]
		instA.Status.ConfigGeneration = "outdated"
		Expect(fakeClient.Status().Update(ctx, instA)).To(Succeed())

		inInstB, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		op.Context().Parent = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		outdated, err := val.OutdatedImports(ctx, inInstB)
		Expect(err).To(Succeed())
		Expect(outdated).To(BeTrue())
	})

	It("should return that no imports are outdated", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstB, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		op.Context().Parent = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		outdated, err := val.OutdatedImports(ctx, inInstB)
		Expect(err).To(Succeed())
		Expect(outdated).To(BeFalse())
	})

	Context("import from manual provided data import", func() {
		It("should report an outdated import", func() {
			ctx := context.Background()
			defer ctx.Done()

			instRoot := fakeInstallations["test1/root"]
			instRoot.Status.Imports[0].ConfigGeneration = "1"
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op, instRoot)
			Expect(err).ToNot(HaveOccurred())

			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			op.Context().Parent = inInstRoot
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			val := imports.NewValidator(op)
			outdated, err := val.OutdatedImports(ctx, inInstRoot)
			Expect(err).To(Succeed())
			Expect(outdated).To(BeTrue())
		})

		It("should return that no imports are outdated", func() {
			ctx := context.Background()
			defer ctx.Done()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())

			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			op.Context().Parent = inInstRoot
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			val := imports.NewValidator(op)
			outdated, err := val.OutdatedImports(ctx, inInstRoot)
			Expect(err).To(Succeed())
			Expect(outdated).To(BeFalse())
		})
	})

})
