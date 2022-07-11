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
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("ImportExecutions", func() {

	var (
		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		fakeCompRepo      ctf.ComponentResolver
	)

	Load := func(inst string) (context.Context, *installations.Installation) {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations[inst])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
		Expect(err).ToNot(HaveOccurred())
		imps, err := rh.GetImports()
		Expect(err).To(Succeed())
		Expect(rh.ImportsSatisfied()).NotTo(HaveOccurred())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, imps)).To(Succeed())
		return ctx, inInstRoot
	}

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

	It("should extend imports by import executions", func() {
		ctx, inst := Load("test11/root")
		exec := imports.New(op)
		err := exec.Ensure(ctx, inst)
		Expect(err).To(Succeed())
		Expect(inst.Imports["processed"]).To(Equal("mytestvalue(extended)"))
	})

	It("should extend imports incrementally by import executions", func() {
		ctx, inst := Load("test11/multi")
		exec := imports.New(op)
		err := exec.Ensure(ctx, inst)
		Expect(err).To(Succeed())
		Expect(inst.Imports["processed"]).To(Equal("mytestvalue(extended)"))
		Expect(inst.Imports["further"]).To(Equal("mytestvalue(extended)(further)"))
	})

	It("should validate imports by import executions", func() {
		ctx, inst := Load("test11/ok")
		exec := imports.New(op)
		err := exec.Ensure(ctx, inst)
		Expect(err).To(Succeed())
	})

	It("should reject wrong imports by import executions", func() {
		ctx, inst := Load("test11/error")
		exec := imports.New(op)
		err := exec.Ensure(ctx, inst)
		Expect(err).NotTo(Succeed())
		Expect(err.Error()).To(Equal("import validation failed: invalid test data:other"))
		Expect(inst.Info.Status.Conditions[0].Type).To(Equal(lsv1alpha1.ConditionType("ValidateImports")))
		Expect(inst.Info.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionStatus("False")))
		Expect(inst.Info.Status.Conditions[0].Reason).To(Equal("ImportValidationFailed"))
		Expect(inst.Info.Status.Conditions[0].Message).To(Equal("invalid test data:other"))
	})
})
