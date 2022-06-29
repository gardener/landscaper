// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper_test

import (
	"context"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Validation", func() {

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
		fakeClient, state, err = envtest.NewFakeClientFromPath("../imports/testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespaces(fakeClient)
		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(fakeCompRepo),
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
				Expect(rh.ImportsSatisfied()).To(Succeed())
			})

			It("should succeed if the import comes from a sibling", func() {
				ctx := context.Background()
				inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
				Expect(err).ToNot(HaveOccurred())
				inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
				Expect(fakeClient.Status().Update(ctx, inInstRoot.Info)).To(Succeed())

				inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
				Expect(err).ToNot(HaveOccurred())
				inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
				Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

				inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstB
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				Expect(rh.ImportsSatisfied()).To(Succeed())
			})

		})

		Context("Target Import", func() {

			It("should succeed if the import comes from a sibling", func() {
				ctx := context.Background()

				inInstE, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/e"])
				Expect(err).ToNot(HaveOccurred())
				inInstE.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

				inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/f"])
				Expect(err).ToNot(HaveOccurred())
				op.Inst = inInstF
				Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

				Expect(op.SetInstallationContext(ctx)).To(Succeed())

				rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
				Expect(err).ToNot(HaveOccurred())
				Expect(rh.ImportsSatisfied()).To(Succeed())
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
				Expect(rh.ImportsSatisfied()).ToNot(Succeed())
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
				Expect(rh.ImportsSatisfied()).ToNot(Succeed())
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
			Expect(rh.ImportsSatisfied()).ToNot(Succeed())
		})

		It("should fail if the parent provides the import but is not progressing", func() {
			if utils.NewReconcile {
				return
			}

			ctx := context.Background()

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseInit
			Expect(fakeClient.Status().Update(ctx, inInstRoot.Info))

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			Expect(rh.ImportsSatisfied()).ToNot(Succeed())
		})

	})

	Context("InstallationsDependingOnReady", func() {

		It("should succeed if all installations which is depended on are ready", func() {
			ctx := context.Background()

			inInstE, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/e"])
			Expect(err).ToNot(HaveOccurred())
			inInstE.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

			inInstF, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/f"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstF
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())

			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())

			Expect(rh.InstallationsDependingOnReady(dependedOnSiblings)).To(Succeed())
		})

		It("should fail if the parent installation is not running", func() {
			ctx := context.Background()

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseInit
			Expect(fakeClient.Status().Update(ctx, inInstRoot.Info))

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("parent installation %q is not progressing", kutil.ObjectKeyFromObject(inInstRoot.Info).String()))
		})

		It("should fail if a root installation's sibling which is depended on is not ready", func() {
			ctx := context.Background()

			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test3/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("depending on installation %q which is not succeeded", kutil.ObjectKey("root-dep", "test3").String()))
		})

		It("should fail if a installation which is depended on is 'Failed'", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseFailed
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			inInstB.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstB.Info)).To(Succeed())

			inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
			Expect(err).ToNot(HaveOccurred())
			inInstC.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstC.Info)).To(Succeed())

			inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstD

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("depending on installation %q which is not succeeded", kutil.ObjectKeyFromObject(inInstA.Info).String()))
		})

		It("should fail if a installation which is depended on is 'Progressing'", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			inInstB.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstB.Info)).To(Succeed())

			inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
			Expect(err).ToNot(HaveOccurred())
			inInstC.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstC.Info)).To(Succeed())

			inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstD

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("depending on installation %q which is not succeeded", kutil.ObjectKeyFromObject(inInstA.Info).String()))
		})

		It("should fail if a installation which is depended on is outdated", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			inInstA.Info.Status.ObservedGeneration = -1
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			inInstB.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstB.Info)).To(Succeed())

			inInstC, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/c"])
			Expect(err).ToNot(HaveOccurred())
			inInstC.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstC.Info)).To(Succeed())

			inInstD, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/d"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstD

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("depending on installation %q which is not up-to-date", kutil.ObjectKeyFromObject(inInstA.Info).String()))
		})

		It("should fail if a installation which is depended on has a reconcile annotation", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
			Expect(fakeClient.Status().Update(ctx, inInstRoot.Info)).To(Succeed())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			lsv1alpha1helper.SetOperation(&inInstA.Info.ObjectMeta, lsv1alpha1.ReconcileOperation)
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstB
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("depending on installation %q which has (force-)reconcile annotation", kutil.ObjectKeyFromObject(inInstA.Info).String()))
		})

		It("should fail if a installation which is depended on has a force-reconcile annotation", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
			Expect(fakeClient.Status().Update(ctx, inInstRoot.Info)).To(Succeed())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			lsv1alpha1helper.SetOperation(&inInstA.Info.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstB
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
			Expect(err).ToNot(HaveOccurred())
			dependedOnSiblings, err := rh.FetchDependencies()
			Expect(err).ToNot(HaveOccurred())
			err = rh.InstallationsDependingOnReady(dependedOnSiblings)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("depending on installation %q which has (force-)reconcile annotation", kutil.ObjectKeyFromObject(inInstA.Info).String()))
		})

	})

	Context("ImportsUpToDate", func() {

		It("should succeed if all imports are up-to-date", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
			Expect(err).ToNot(HaveOccurred())
			inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstRoot.Info)).To(Succeed())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

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
			inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			Expect(fakeClient.Status().Update(ctx, inInstRoot.Info)).To(Succeed())

			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded
			inInstA.Info.Status.ConfigGeneration = "updated"
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())

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

	Context("InstUpToDate", func() {

		It("should succeed if the installation is up-to-date", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstA

			Expect(reconcilehelper.InstUpToDate(op.Inst.Info)).To(BeTrue())
		})

		It("should fail if the installation is not up-to-date", func() {
			ctx := context.Background()
			inInstA, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/a"])
			Expect(err).ToNot(HaveOccurred())
			inInstA.Info.Status.ObservedGeneration = -1
			Expect(fakeClient.Status().Update(ctx, inInstA.Info)).To(Succeed())
			op.Inst = inInstA

			Expect(reconcilehelper.InstUpToDate(op.Inst.Info)).To(BeFalse())
		})

	})

})
