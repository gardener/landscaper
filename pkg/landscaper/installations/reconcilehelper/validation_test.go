// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Validation", func() {

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

		registryAccess, err := registries.GetFactory().NewLocalRegistryAccess("../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(registryAccess),
		}
	})

	Context("ImportsSatisfied", func() {

		Context("Data Import", func() {

			It("should succeed if the import comes from the parent", func() {
				ctx := context.Background()

				inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstA
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				_, err = rh.ImportsSatisfied()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should succeed if the import comes from a sibling", func() {
				ctx := context.Background()
				inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeClient.Status().Update(ctx, inInstRoot.GetInstallation())).To(Succeed())

				inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeClient.Status().Update(ctx, inInstA.GetInstallation())).To(Succeed())

				inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstB
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				_, err = rh.ImportsSatisfied()
				Expect(err).ToNot(HaveOccurred())
			})

		})

		Context("Target Import", func() {

			It("should succeed if the import comes from a sibling", func() {
				ctx := context.Background()

				_, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/e"])
				Expect(err).ToNot(HaveOccurred())

				inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/f"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstF
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				_, err = rh.ImportsSatisfied()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail if a target import from a manually added target is not present", func() {
				ctx := context.Background()
				inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstRoot
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				target := &lsv1alpha1.Target{}
				target.Name = lsv1alpha1helper.GenerateDataObjectName("", "ext.a")
				target.Namespace = "test4"
				Expect(fakeClient.Delete(ctx, target))

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				_, err = rh.ImportsSatisfied()
				Expect(err).To(HaveOccurred())
			})

			It("should fail if a import from a parent import is not present", func() {
				ctx := context.Background()
				inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/f"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstF
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				target := &lsv1alpha1.Target{}
				target.Name = lsv1alpha1helper.GenerateDataObjectName(op.Context().Name, "root.a")
				target.Namespace = "test4"
				Expect(fakeClient.Delete(ctx, target))

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				_, err = rh.ImportsSatisfied()
				Expect(err).To(HaveOccurred())
			})

		})

		It("should fail if neither the parent nor a sibling provide the import", func() {
			ctx := context.Background()

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test11/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			_, err = rh.ImportsSatisfied()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("InstallationsDependingOnReady", func() {

		It("should succeed if all installations which is depended on are ready", func() {
			ctx := context.Background()

			inInstE, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/e"])
			Expect(err).ToNot(HaveOccurred())
			inInstE.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded

			inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/f"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstF
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())

			predecessors := rh.FetchPredecessors()

			predecessorMap, err := rh.GetPredecessors(inInstE.GetInstallation(), predecessors)
			Expect(err).ToNot(HaveOccurred())

			err = rh.AllPredecessorsFinished(inInstE.GetInstallation(), predecessorMap)
			Expect(err).ToNot(HaveOccurred())

			err = rh.AllPredecessorsSucceeded(inInstE.GetInstallation(), predecessorMap)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail if a preceding installation is 'Failed'", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Failed
			Expect(fakeClient.Status().Update(ctx, inInstA.GetInstallation())).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			inInstB.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Failed
			Expect(fakeClient.Status().Update(ctx, inInstB.GetInstallation())).To(Succeed())

			inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
			Expect(err).ToNot(HaveOccurred())
			inInstC.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded
			Expect(fakeClient.Status().Update(ctx, inInstC.GetInstallation())).To(Succeed())

			inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstD

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())

			predecessors := rh.FetchPredecessors()
			Expect(err).ToNot(HaveOccurred())

			predecessorMap, err := rh.GetPredecessors(inInstD.GetInstallation(), predecessors)
			Expect(err).ToNot(HaveOccurred())

			err = rh.AllPredecessorsFinished(inInstD.GetInstallation(), predecessorMap)
			Expect(err).ToNot(HaveOccurred())

			err = rh.AllPredecessorsSucceeded(inInstD.GetInstallation(), predecessorMap)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
		})

		It("should fail if a preceding installation is 'Progressing'", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Progressing
			inInstA.GetInstallation().Status.JobID = "2"
			inInstA.GetInstallation().Status.JobIDFinished = "1"
			Expect(fakeClient.Status().Update(ctx, inInstA.GetInstallation())).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			inInstB.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Progressing
			inInstB.GetInstallation().Status.JobID = "2"
			inInstB.GetInstallation().Status.JobIDFinished = "1"
			Expect(fakeClient.Status().Update(ctx, inInstB.GetInstallation())).To(Succeed())

			inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
			Expect(err).ToNot(HaveOccurred())
			inInstC.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded
			inInstC.GetInstallation().Status.JobID = "2"
			inInstC.GetInstallation().Status.JobIDFinished = "2"
			Expect(fakeClient.Status().Update(ctx, inInstC.GetInstallation())).To(Succeed())

			inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
			Expect(err).ToNot(HaveOccurred())
			inInstD.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Init
			inInstD.GetInstallation().Status.JobID = "2"
			inInstD.GetInstallation().Status.JobIDFinished = "1"
			Expect(fakeClient.Status().Update(ctx, inInstB.GetInstallation())).To(Succeed())

			op.Inst = inInstD
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())

			predecessors := rh.FetchPredecessors()
			Expect(err).ToNot(HaveOccurred())

			predecessorMap, err := rh.GetPredecessors(inInstD.GetInstallation(), predecessors)
			Expect(err).ToNot(HaveOccurred())

			err = rh.AllPredecessorsFinished(inInstD.GetInstallation(), predecessorMap)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
		})

		It("should fail if a preceding installation is outdated", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded
			inInstA.GetInstallation().Status.JobID = "1"
			inInstA.GetInstallation().Status.JobIDFinished = "1"
			Expect(fakeClient.Status().Update(ctx, inInstA.GetInstallation())).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			inInstB.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded
			inInstB.GetInstallation().Status.JobID = "1"
			inInstB.GetInstallation().Status.JobIDFinished = "1"
			Expect(fakeClient.Status().Update(ctx, inInstB.GetInstallation())).To(Succeed())

			inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
			Expect(err).ToNot(HaveOccurred())
			inInstC.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded
			inInstC.GetInstallation().Status.JobID = "2"
			inInstC.GetInstallation().Status.JobIDFinished = "2"
			Expect(fakeClient.Status().Update(ctx, inInstC.GetInstallation())).To(Succeed())

			inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
			Expect(err).ToNot(HaveOccurred())
			inInstD.GetInstallation().Status.InstallationPhase = lsv1alpha1.InstallationPhases.Init
			inInstD.GetInstallation().Status.JobID = "2"
			inInstD.GetInstallation().Status.JobIDFinished = "1"
			Expect(fakeClient.Status().Update(ctx, inInstC.GetInstallation())).To(Succeed())

			op.Inst = inInstD
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())

			predecessors := rh.FetchPredecessors()

			predecessorMap, err := rh.GetPredecessors(inInstD.GetInstallation(), predecessors)
			Expect(err).ToNot(HaveOccurred())

			err = rh.AllPredecessorsFinished(inInstD.GetInstallation(), predecessorMap)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
		})
	})

	Context("ImportsUpToDate", func() {

		It("should succeed if all imports are up-to-date", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeClient.Status().Update(ctx, inInstRoot.GetInstallation())).To(Succeed())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeClient.Status().Update(ctx, inInstA.GetInstallation())).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstB
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			utd, err := rh.ImportsUpToDate()
			Expect(err).ToNot(HaveOccurred())
			Expect(utd).To(BeTrue())
		})

		It("should fail if an import has changed", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeClient.Status().Update(ctx, inInstRoot.GetInstallation())).To(Succeed())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.GetInstallation().Status.ConfigGeneration = "updated"
			Expect(fakeClient.Status().Update(ctx, inInstA.GetInstallation())).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstB
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			utd, err := rh.ImportsUpToDate()
			Expect(err).ToNot(HaveOccurred())
			Expect(utd).To(BeFalse())
		})

	})

})
