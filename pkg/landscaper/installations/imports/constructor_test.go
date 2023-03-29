// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils"
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

		createDefaultContextsForNamespace(fakeClient)
		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient("../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		registry := cnudie.NewRegistry(fakeCompRepo)
		op = &installations.Operation{
			Operation: lsoperation.NewOperation(fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(registry),
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
		Expect(c.Construct(ctx, nil)).To(Succeed())
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
		Expect(c.Construct(ctx, nil)).To(Succeed())
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
		Expect(c.Construct(ctx, nil)).To(Succeed())
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
		Expect(c.Construct(ctx, nil)).To(Succeed())
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
		Expect(c.Construct(ctx, nil)).To(Succeed())
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

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			do := &lsv1alpha1.DataObject{}
			do.Name = lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromInstallation(inInstRoot.GetInstallation()), "root.a")
			do.Namespace = inInstRoot.GetInstallation().Namespace
			Expect(fakeClient.Get(ctx, kutil.ObjectKey(do.Name, do.Namespace), do)).To(Succeed())
			do.Data.RawMessage = []byte("7")
			Expect(fakeClient.Update(ctx, do)).To(Succeed())

			c := imports.NewConstructor(op)
			err = c.Construct(ctx, nil)
			Expect(installations.IsSchemaValidationFailedError(err)).To(BeTrue())
		})

		It("should handle missing schema definition in import gracefully", func() {
			ctx := context.Background()
			inInstJ, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test12/j"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstJ
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			c := imports.NewConstructor(op)
			Expect(c.Construct(ctx, nil)).ToNot(Succeed())
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
			Expect(c.Construct(ctx, nil)).To(Succeed())
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
			Expect(c.Construct(ctx, nil)).To(Succeed())
			Expect(inInstF.GetImports()).ToNot(BeNil())

			Expect(inInstF.GetImports()).To(HaveKeyWithValue("f.a", MatchKeys(IgnoreExtras, Keys{
				"spec": MatchKeys(IgnoreExtras, Keys{
					"type":   Equal("landscaper.gardener.cloud/mock"),
					"config": Equal("val-e"),
				}),
			})))
		})
	})

	Context("TargetLists", func() {
		It("should construct a targetlist import from target names", func() {
			ctx := context.Background()
			defer ctx.Done()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test9/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			utils.ExpectNoError(op.ResolveComponentDescriptors(ctx))
			utils.ExpectNoError(op.SetInstallationContext(ctx))

			c := imports.NewConstructor(op)
			utils.ExpectNoError(c.Construct(ctx, nil))
			Expect(inInstRoot.GetImports()).ToNot(BeNil())

			Expect(inInstRoot.GetImports()).To(HaveKeyWithValue("root.a", ConsistOf(
				MatchKeys(IgnoreExtras, Keys{
					"spec": MatchKeys(IgnoreExtras, Keys{
						"type":   Equal("landscaper.gardener.cloud/mock"),
						"config": Equal("val-ext-a1"),
					}),
				}),
				MatchKeys(IgnoreExtras, Keys{
					"spec": MatchKeys(IgnoreExtras, Keys{
						"type":   Equal("landscaper.gardener.cloud/mock"),
						"config": Equal("val-ext-a2"),
					}),
				}),
				MatchKeys(IgnoreExtras, Keys{
					"spec": MatchKeys(IgnoreExtras, Keys{
						"type":   Equal("landscaper.gardener.cloud/mock"),
						"config": Equal("val-sib-a"),
					}),
				}),
			)))
		})
	})

})
