// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/utils/landscaper"

	"github.com/gardener/landscaper/apis/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserror "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("ConditionalImports", func() {

	var (
		ctx  context.Context
		octx ocm.Context

		op *installations.Operation

		instRef types.NamespacedName
		cmRef   lsv1alpha1.ObjectReference

		fakeClient client.Client
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		var err error

		instRef = kutil.ObjectKey("conditional-import-inst", "test8")
		cmRef = lsv1alpha1.ObjectReference{
			Name:      "inst-import",
			Namespace: instRef.Namespace,
		}

		fakeClient, _, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespace(fakeClient)

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "../testdata/registry"}
		registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, nil, nil, nil, nil, localregistryconfig, nil, nil)
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

	It("should remove imports based on optional/conditional parent imports from subinstallation", func() {
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
		utils.ExpectNoError(subInstOp.Ensure(ctx, nil))
		subinsts, err := landscaper.GetSubInstallationsOfInstallation(ctx, fakeClient, conInst.GetInstallation())
		utils.ExpectNoError(err)
		Expect(len(subinsts) > 0).To(BeTrue())

		subinst := &lsv1alpha1.Installation{}
		found := false
		for i := range subinsts { // fetch subinstallation from client
			name := subinsts[i].Annotations[lsv1alpha1.SubinstallationNameAnnotation]
			if name == "subinst-import" {
				subinst = subinsts[i]
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
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(fakeClient.Get(ctx, instRef, inst))
		// add imports to installation
		inst.Spec.Imports.Data = append(inst.Spec.Imports.Data, lsv1alpha1.DataImport{
			Name: "rootcond.foo",
			ConfigMapRef: &lsv1alpha1.LocalConfigMapReference{
				Key:  "foo",
				Name: cmRef.Name,
			},
		}, lsv1alpha1.DataImport{
			Name: "rootcond.bar",
			ConfigMapRef: &lsv1alpha1.LocalConfigMapReference{
				Key:  "bar",
				Name: cmRef.Name,
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
		utils.ExpectNoError(subInstOp.Ensure(ctx, nil))
		subinsts, err := landscaper.GetSubInstallationsOfInstallation(ctx, fakeClient, conInst.GetInstallation())
		utils.ExpectNoError(err)
		Expect(len(subinsts) > 0).To(BeTrue())

		subinst := &lsv1alpha1.Installation{}
		found := false
		for i := range subinsts { // fetch subinstallation from client
			name := subinsts[i].Annotations[lsv1alpha1.SubinstallationNameAnnotation]
			if name == "subinst-import" {
				subinst = subinsts[i]
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
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(fakeClient.Get(ctx, instRef, inst))
		// add imports to installation
		inst.Spec.Imports.Data = append(inst.Spec.Imports.Data, lsv1alpha1.DataImport{
			Name: "rootcond.foo",
			ConfigMapRef: &lsv1alpha1.LocalConfigMapReference{
				Key:  "foo",
				Name: cmRef.Name,
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
