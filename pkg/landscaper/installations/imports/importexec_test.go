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
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("ImportExecutions", func() {

	var (
		ctx  context.Context
		octx ocm.Context

		op *installations.Operation
		c  *imports.Constructor

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
	)

	Load := func(ctx context.Context, inst string) *imports.Constructor {
		inInstRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations[inst],
			op.LsUncachedClient(), op.ComponentsRegistry())
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
		Expect(err).ToNot(HaveOccurred())
		imps, err := rh.ImportsSatisfied(ctx)
		Expect(err).To(Succeed())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, imps)).To(Succeed())
		return c
	}

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

	It("should extend imports by import executions", func() {
		c = Load(ctx, "test11/root")
		err := c.RenderImportExecutions()
		Expect(err).To(Succeed())
		Expect(c.Inst.GetImports()["processed"]).To(Equal("mytestvalue(extended)"))
	})

	It("should extend imports incrementally by import executions", func() {
		c = Load(ctx, "test11/multi")
		err := c.RenderImportExecutions()
		Expect(err).To(Succeed())
		Expect(c.Inst.GetImports()["processed"]).To(Equal("mytestvalue(extended)"))
		Expect(c.Inst.GetImports()["further"]).To(Equal("mytestvalue(extended)(further)"))
	})

	It("should validate imports by import executions", func() {
		c = Load(ctx, "test11/ok")
		err := c.RenderImportExecutions()
		Expect(err).To(Succeed())
	})

	It("should reject wrong imports by import executions", func() {
		c = Load(ctx, "test11/error")
		err := c.RenderImportExecutions()
		Expect(err).NotTo(Succeed())
		Expect(err.Error()).To(Equal("import validation failed: invalid test data:other"))
		Expect(c.Inst.GetInstallation().Status.Conditions[0].Type).To(Equal(lsv1alpha1.ConditionType("ValidateImports")))
		Expect(c.Inst.GetInstallation().Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionStatus("False")))
		Expect(c.Inst.GetInstallation().Status.Conditions[0].Reason).To(Equal("ImportValidationFailed"))
		Expect(c.Inst.GetInstallation().Status.Conditions[0].Message).To(Equal("invalid test data:other"))
	})
})
