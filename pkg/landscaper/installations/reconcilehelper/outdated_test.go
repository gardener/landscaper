// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper_test

import (
	"context"

	"github.com/gardener/landscaper/apis/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("OutdatedImports", func() {

	var (
		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
	)

	BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("../imports/testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespaces(fakeClient)
		fakeInstallations = state.Installations

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "../testdata/registry"}
		registryAccess, err := registries.GetFactory().NewRegistryAccess(context.Background(), nil, nil, nil, localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		operation, err := lsoperation.NewBuilder().Client(fakeClient).Scheme(api.LandscaperScheme).WithEventRecorder(record.NewFakeRecorder(1024)).ComponentRegistry(registryAccess).Build(context.Background())
		Expect(err).ToNot(HaveOccurred())
		op = &installations.Operation{
			Operation: operation,
		}
	})

	It("should return that imports are outdated if a import from the parent is outdated", func() {
		ctx := context.Background()

		inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
		Expect(err).ToNot(HaveOccurred())
		upToDate, err := rh.ImportsUpToDate()
		Expect(err).To(Succeed())
		Expect(upToDate).To(BeFalse())
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

		rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
		Expect(err).ToNot(HaveOccurred())
		upToDate, err := rh.ImportsUpToDate()
		Expect(err).To(Succeed())
		Expect(upToDate).To(BeFalse())
	})

	It("should return that no imports are outdated", func() {
		ctx := context.Background()

		inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
		Expect(err).ToNot(HaveOccurred())
		upToDate, err := rh.ImportsUpToDate()
		Expect(err).To(Succeed())
		Expect(upToDate).To(BeTrue())
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

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			upToDate, err := rh.ImportsUpToDate()
			Expect(err).To(Succeed())
			Expect(upToDate).To(BeFalse())
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

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			upToDate, err := rh.ImportsUpToDate()
			Expect(err).To(Succeed())
			Expect(upToDate).To(BeTrue())
		})
	})

})
