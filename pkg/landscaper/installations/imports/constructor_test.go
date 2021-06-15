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
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Constructor", func() {

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

		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme).
				SetComponentsRegistry(fakeCompRepo),
		}
	})

	It("should construct the imported config from a sibling", func() {
		ctx := context.Background()
		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test2/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"b.a": "val-a",
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstB)).To(Succeed())
		Expect(inInstB.GetImports()).ToNot(BeNil())
		Expect(inInstB.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a sibling and the indirect parent import", func() {
		ctx := context.Background()
		inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test2/c"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstC
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"c.a": "val-a",
			"c.b": "val-root-import", // from root import
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstC)).To(Succeed())
		Expect(inInstC.GetImports()).ToNot(BeNil())

		Expect(inInstC.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a manual created data object", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test5/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"root.a": "val-root-import",
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())

		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a secret", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test6/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"root.a": "val-root-import",
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())
		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a configmap", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test7/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"root.a": "val-root-import",
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())
		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	Context("schema validation", func() {
		It("should forbid when the import of a component does not satisfy the schema", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			op.Context().Parent = inInstRoot
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			do := &lsv1alpha1.DataObject{}
			do.Name = lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromInstallation(inInstRoot.Info), "root.a")
			do.Namespace = inInstRoot.Info.Namespace
			Expect(fakeClient.Get(ctx, kutil.ObjectKey(do.Name, do.Namespace), do)).To(Succeed())
			do.Data.RawMessage = []byte("7")
			Expect(fakeClient.Update(ctx, do)).To(Succeed())

			c := imports.NewConstructor(op)
			err = c.Construct(ctx, inInstA)
			Expect(installations.IsSchemaValidationFailedError(err)).To(BeTrue())
		})
	})

	Context("Targets", func() {
		It("should construct import from a manually added target", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			c := imports.NewConstructor(op)
			Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
			Expect(inInstRoot.GetImports()).ToNot(BeNil())

			Expect(inInstRoot.GetImports()).To(HaveKeyWithValue("root.a", MatchKeys(IgnoreExtras, Keys{
				"spec": MatchKeys(IgnoreExtras, Keys{
					"type":   Equal("landscaper.gardener.cloud/mock"),
					"config": Equal("val-e"),
				}),
			})))
		})

		It("should construct import from a parent import", func() {
			ctx := context.Background()
			inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/f"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstF
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			c := imports.NewConstructor(op)
			Expect(c.Construct(ctx, inInstF)).To(Succeed())
			Expect(inInstF.GetImports()).ToNot(BeNil())

			Expect(inInstF.GetImports()).To(HaveKeyWithValue("f.a", MatchKeys(IgnoreExtras, Keys{
				"spec": MatchKeys(IgnoreExtras, Keys{
					"type":   Equal("landscaper.gardener.cloud/mock"),
					"config": Equal("val-e"),
				}),
			})))
		})
	})

})
