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
	"github.com/gardener/landscaper/pkg/api"
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

		createDefaultContextsForNamespace(fakeClient)
		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(fakeCompRepo),
		}
	})

	It("should return that imports are outdated if a import from the parent is outdated", func() {
		ctx := context.Background()

		inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		val, err := imports.NewValidator(ctx, op)
		Expect(err).To(Succeed())

		outdated, err := val.OutdatedImports(ctx)
		Expect(err).To(Succeed())
		Expect(outdated).To(BeTrue())
	})

	It("should return that imports are outdated if a import from another component is outdated", func() {
		ctx := context.Background()

		instA := fakeInstallations["test1/a"]
		instA.Status.ConfigGeneration = "outdated"
		Expect(fakeClient.Status().Update(ctx, instA)).To(Succeed())

		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		val, err := imports.NewValidator(ctx, op)
		Expect(err).To(Succeed())

		outdated, err := val.OutdatedImports(ctx)
		Expect(err).To(Succeed())
		Expect(outdated).To(BeTrue())
	})

	It("should return that no imports are outdated", func() {
		ctx := context.Background()

		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		val, err := imports.NewValidator(ctx, op)
		Expect(err).To(Succeed())

		outdated, err := val.OutdatedImports(ctx)
		Expect(err).To(Succeed())
		Expect(outdated).To(BeFalse())
	})

	Context("import from manual provided data import", func() {
		It("should report an outdated import", func() {
			ctx := context.Background()

			instRoot := fakeInstallations["test1/root"]
			instRoot.Status.Imports[0].ConfigGeneration = "1"
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), instRoot)
			Expect(err).ToNot(HaveOccurred())

			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())
			val, err := imports.NewValidator(ctx, op)
			Expect(err).To(Succeed())

			outdated, err := val.OutdatedImports(ctx)
			Expect(err).To(Succeed())
			Expect(outdated).To(BeTrue())
		})

		It("should return that no imports are outdated", func() {
			ctx := context.Background()
			instRoot := fakeInstallations["test1/root"]
			instRoot.Status.Imports[0].ConfigGeneration = "d62a6724cde91ae43d5946a9dcb581b873194b74"
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), instRoot)
			Expect(err).ToNot(HaveOccurred())

			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())
			val, err := imports.NewValidator(ctx, op)
			Expect(err).To(Succeed())

			outdated, err := val.OutdatedImports(ctx)
			Expect(err).To(Succeed())
			Expect(outdated).To(BeFalse())
		})
	})

})
