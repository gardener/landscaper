// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/apis/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Constructor", func() {

	var (
		ctx  context.Context
		octx ocm.Context

		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespace(fakeClient)
		fakeInstallations = state.Installations

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "../testdata/registry"}
		registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, nil, nil, nil,
			localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		operation, err := lsoperation.NewBuilder().WithLsUncachedClient(fakeClient).Scheme(api.LandscaperScheme).WithEventRecorder(record.NewFakeRecorder(1024)).ComponentRegistry(registryAccess).Build(ctx)
		Expect(err).ToNot(HaveOccurred())
		op = &installations.Operation{
			Operation: operation,
		}
	})

	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	It("should construct the imported config from a sibling", func() {
		inInstB, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test2/b"],
			op.LsUncachedClient(), op.ComponentsRegistry())
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
		inInstC, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test2/c"],
			op.LsUncachedClient(), op.ComponentsRegistry())
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
		inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test5/root"],
			op.LsUncachedClient(), op.ComponentsRegistry())
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
		inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test6/root"],
			op.LsUncachedClient(), op.ComponentsRegistry())
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
		inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test7/root"],
			op.LsUncachedClient(), op.ComponentsRegistry())
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

	It("should use defaults defined in blueprint for missing optional imports", func() {
		inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test13/root"],
			op.LsUncachedClient(), op.ComponentsRegistry())
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"defaulted": map[string]interface{}{
				"foo": "bar",
			},
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, nil)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())
		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	Context("schema validation", func() {
		It("should forbid when the import of a component does not satisfy the schema", func() {
			inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test1/root"],
				op.LsUncachedClient(), op.ComponentsRegistry())
			Expect(err).ToNot(HaveOccurred())

			inInstA, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test1/a"],
				op.LsUncachedClient(), op.ComponentsRegistry())
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
			inInstJ, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test12/j"],
				op.LsUncachedClient(), op.ComponentsRegistry())
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
			inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test4/root"],
				op.LsUncachedClient(), op.ComponentsRegistry())
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
			inInstF, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test4/f"],
				op.LsUncachedClient(), op.ComponentsRegistry())
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
			inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test9/root"],
				op.LsUncachedClient(), op.ComponentsRegistry())
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
