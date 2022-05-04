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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lserror "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("ConditionalImports", func() {

	var (
		op *installations.Operation

		instRef types.NamespacedName
		cmRef   lsv1alpha1.ObjectReference

		fakeClient   client.Client
		fakeCompRepo ctf.ComponentResolver
	)

	BeforeEach(func() {
		var err error

		instRef = kutil.ObjectKey("conditional-import-inst", "test8")
		cmRef = lsv1alpha1.ObjectReference{
			Name:      "inst-import",
			Namespace: instRef.Namespace,
		}

		fakeClient, _, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespace(fakeClient)

		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(fakeCompRepo),
		}
	})

	It("should remove imports based on optional/conditional parent imports from subinstallation", func() {
		ctx := context.Background()
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(fakeClient.Get(ctx, instRef, inst))
		conInst, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), inst)
		utils.ExpectNoError(err)
		op.Inst = conInst
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op.Operation, conInst, nil)
		utils.ExpectNoError(err)
		subInstOp := subinstallations.New(instOp)
		// satisfy imports
		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		Expect(imports.NewConstructor(op).Construct(ctx, nil)).To(Succeed())
		// create subinstallation
		utils.ExpectNoError(subInstOp.Ensure(ctx))
		Expect(conInst.Info.Status.InstallationReferences).NotTo(BeEmpty())
		subinst := &lsv1alpha1.Installation{}
		found := false
		for _, sir := range conInst.Info.Status.InstallationReferences { // fetch subinstallation from client
			if sir.Name == "subinst-import" {
				utils.ExpectNoError(fakeClient.Get(ctx, sir.Reference.NamespacedName(), subinst))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "subinstallation not found in subinstallation references")

		// parent installation has no imports, therefore only the sibling import should exist
		Expect(subinst.Spec.Imports.Data).To(HaveLen(1))
		Expect(subinst.Spec.Imports.Data).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"DataRef": Equal("exp.baz"),
			"Name":    Equal("internalBaz"),
		})))
	})

	It("should not remove imports based on optional/conditional parent imports which are satisfied from subinstallation", func() {
		ctx := context.Background()
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(fakeClient.Get(ctx, instRef, inst))
		// add imports to installation
		inst.Spec.Imports.Data = append(inst.Spec.Imports.Data, lsv1alpha1.DataImport{
			Name: "rootcond.foo",
			ConfigMapRef: &lsv1alpha1.ConfigMapReference{
				Key:             "foo",
				ObjectReference: cmRef,
			},
		}, lsv1alpha1.DataImport{
			Name: "rootcond.bar",
			ConfigMapRef: &lsv1alpha1.ConfigMapReference{
				Key:             "bar",
				ObjectReference: cmRef,
			},
		})
		utils.ExpectNoError(fakeClient.Update(ctx, inst))
		conInst, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), inst)
		utils.ExpectNoError(err)
		op.Inst = conInst
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op.Operation, conInst, nil)
		utils.ExpectNoError(err)
		subInstOp := subinstallations.New(instOp)
		// satisfy imports
		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		Expect(imports.NewConstructor(op).Construct(ctx, nil)).To(Succeed())
		// create subinstallation
		utils.ExpectNoError(subInstOp.Ensure(ctx))
		Expect(conInst.Info.Status.InstallationReferences).NotTo(BeEmpty())
		subinst := &lsv1alpha1.Installation{}
		found := false
		for _, sir := range conInst.Info.Status.InstallationReferences { // fetch subinstallation from client
			if sir.Name == "subinst-import" {
				utils.ExpectNoError(fakeClient.Get(ctx, sir.Reference.NamespacedName(), subinst))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "subinstallation not found in subinstallation references")

		// parent installation has optional/conditional imports satisfied, therefore no imports should have been deleted from the subinstallation
		Expect(subinst.Spec.Imports.Data).To(HaveLen(3))
		Expect(subinst.Spec.Imports.Data).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"DataRef": Equal("rootcond.foo"),
			"Name":    Equal("internalFoo"),
		})))
		Expect(subinst.Spec.Imports.Data).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"DataRef": Equal("rootcond.bar"),
			"Name":    Equal("internalBar"),
		})))
		Expect(subinst.Spec.Imports.Data).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"DataRef": Equal("exp.baz"),
			"Name":    Equal("internalBaz"),
		})))
	})

	It("should not succeed if a conditional import is not fulfilled while it's condition is fulfilled", func() {
		ctx := context.Background()
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(fakeClient.Get(ctx, instRef, inst))
		// add imports to installation
		inst.Spec.Imports.Data = append(inst.Spec.Imports.Data, lsv1alpha1.DataImport{
			Name: "rootcond.foo",
			ConfigMapRef: &lsv1alpha1.ConfigMapReference{
				Key:             "foo",
				ObjectReference: cmRef,
			},
		})
		utils.ExpectNoError(fakeClient.Update(ctx, inst))
		conInst, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), inst)
		utils.ExpectNoError(err)
		op.Inst = conInst
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
		// satisfy imports
		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		err = imports.NewConstructor(op).Construct(ctx, nil)
		parsedErr, ok := err.(lserror.LsError)
		Expect(ok).To(BeTrue(), "error should be of type installations.Error")
		Expect(installations.IsImportNotFoundError(parsedErr)).To(BeTrue())
		Expect(parsedErr.LandscaperError().Message).To(ContainSubstring("rootcond.bar"))
	})

})
