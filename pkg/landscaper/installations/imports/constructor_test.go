// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"
	"encoding/json"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	testutils "github.com/gardener/landscaper/test/utils"
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

		createDefaultContextsForNamespace(fakeClient)

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(fakeCompRepo),
		}
	})

	It("should construct the imported config from a sibling", func() {
		ctx := context.Background()
		inInstB, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test2/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"b.a": "val-a",
		}

		Expect(op.SetInstallationScope(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstB)).To(Succeed())
		Expect(inInstB.GetImports()).ToNot(BeNil())
		Expect(inInstB.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a sibling and the indirect parent import", func() {
		ctx := context.Background()
		inInstC, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test2/c"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstC
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"c.a": "val-a",
			"c.b": "val-root-import", // from root import
		}

		Expect(op.SetInstallationScope(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstC)).To(Succeed())
		Expect(inInstC.GetImports()).ToNot(BeNil())

		Expect(inInstC.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a manual created data object", func() {
		ctx := context.Background()
		inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test5/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"root.a": "val-root-import",
		}

		Expect(op.SetInstallationScope(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())

		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a secret", func() {
		ctx := context.Background()
		inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test6/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"root.a": "val-root-import",
		}

		Expect(op.SetInstallationScope(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())
		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a configmap", func() {
		ctx := context.Background()
		inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test7/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"root.a": "val-root-import",
		}

		Expect(op.SetInstallationScope(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
		Expect(inInstRoot.GetImports()).ToNot(BeNil())
		Expect(inInstRoot.GetImports()).To(Equal(expectedConfig))
	})

	Context("schema validation", func() {
		It("should forbid when the import of a component does not satisfy the schema", func() {
			ctx := context.Background()
			inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())

			inInstA, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			op.Scope().Parent = inInstRoot
			Expect(op.SetInstallationScope(ctx)).To(Succeed())

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
			inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationScope(ctx)).To(Succeed())

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
			inInstF, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/f"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstF
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationScope(ctx)).To(Succeed())

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

	Context("TargetLists", func() {
		It("should construct a targetlist import from target names", func() {
			ctx := context.Background()
			defer ctx.Done()
			inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test9/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			testutils.ExpectNoError(op.ResolveComponentDescriptors(ctx))
			testutils.ExpectNoError(op.SetInstallationScope(ctx))

			c := imports.NewConstructor(op)
			testutils.ExpectNoError(c.Construct(ctx, inInstRoot))
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

	Context("ComponentDescriptors", func() {
		It("should construct component descriptor imports from registry, secret, and configmap", func() {
			ctx := context.Background()
			inInstRoot, err := testutils.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test10/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationScope(ctx)).To(Succeed())

			c := imports.NewConstructor(op)
			Expect(c.Construct(ctx, inInstRoot)).To(Succeed())
			Expect(inInstRoot.GetImports()).ToNot(BeNil())

			// check import from registry
			cdData, err := dataobjects.NewComponentDescriptor().SetDescriptor(op.ComponentDescriptor).GetData()
			testutils.ExpectNoError(err)
			Expect(inInstRoot.GetImports()).To(HaveKeyWithValue("cd-from-registry", BeEquivalentTo(cdData)))

			// check import from configmap
			cm := &k8sv1.ConfigMap{}
			testutils.ExpectNoError(fakeClient.Get(ctx, kutil.ObjectKey("my-cd-configmap", "test10"), cm))
			tmpData := cm.Data["componentDescriptor"]
			tmpDataJSON, err := yaml.ToJSON([]byte(tmpData))
			testutils.ExpectNoError(err)
			configMapCD := &cdv2.ComponentDescriptor{}
			testutils.ExpectNoError(json.Unmarshal(tmpDataJSON, configMapCD))
			configMapCDData, err := dataobjects.NewComponentDescriptor().SetDescriptor(configMapCD).GetData()
			testutils.ExpectNoError(err)
			Expect(inInstRoot.GetImports()).To(HaveKeyWithValue("cd-from-configmap", BeEquivalentTo(configMapCDData)))

			// check import from secret
			secret := &k8sv1.Secret{}
			testutils.ExpectNoError(fakeClient.Get(ctx, kutil.ObjectKey("my-cd-secret", "test10"), secret))
			tmpDataByte := secret.Data["componentDescriptor"]
			tmpDataJSON, err = yaml.ToJSON(tmpDataByte)
			testutils.ExpectNoError(err)
			secretCD := &cdv2.ComponentDescriptor{}
			testutils.ExpectNoError(json.Unmarshal(tmpDataJSON, secretCD))
			secretCDData, err := dataobjects.NewComponentDescriptor().SetDescriptor(secretCD).GetData()
			testutils.ExpectNoError(err)
			Expect(inInstRoot.GetImports()).To(HaveKeyWithValue("cd-from-secret", BeEquivalentTo(secretCDData)))

			// check component descriptor list import
			Expect(inInstRoot.GetImports()).To(HaveKeyWithValue("cdlist", And(
				HaveLen(3),
				ContainElement(cdData),
				ContainElement(configMapCDData),
				ContainElement(secretCDData),
			)))
		})
	})

})
