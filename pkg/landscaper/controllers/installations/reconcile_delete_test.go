// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	testutils "github.com/gardener/landscaper/test/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

	var (
		op   *lsoperation.Operation
		ctrl reconcile.Reconciler

		state        *envtest.State
		fakeCompRepo ctf.ComponentResolver
	)

	BeforeEach(func() {
		var err error
		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "./testdata")
		Expect(err).ToNot(HaveOccurred())

		op = lsoperation.NewOperation(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(fakeCompRepo)

		ctrl = installationsctl.NewTestActuator(*op, &config.LandscaperConfiguration{
			Registry: config.RegistryConfiguration{
				Local: &config.LocalRegistryConfiguration{
					RootPath: "./testdata",
				},
			},
		})
	})

	AfterEach(func() {
		if state != nil {
			ctx := context.Background()
			defer ctx.Done()
			Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
			state = nil
		}
	})

	It("should not delete if another installation still imports a exported value", func() {
		ctx := context.Background()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test1")
		Expect(err).ToNot(HaveOccurred())
		Expect(testenv.InitDefaultContextFromInst(ctx, state, state.Installations[state.Namespace+"/a"]))

		inInstA, err := testutils.CreateInternalInstallation(ctx,
			op.ComponentsRegistry(),
			state.Installations[state.Namespace+"/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstA)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(installationsctl.SiblingImportError))

		instC := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
		Expect(instC.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should block deletion if there are still subinstallations", func() {
		ctx := context.Background()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test1")
		Expect(err).ToNot(HaveOccurred())
		Expect(testenv.InitDefaultContextFromInst(ctx, state, state.Installations[state.Namespace+"/root"]))

		inInstRoot, err := testutils.CreateInternalInstallation(ctx,
			op.ComponentsRegistry(),
			state.Installations[state.Namespace+"/root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstRoot)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		instA := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "a", Namespace: state.Namespace}, instA)).ToNot(HaveOccurred())
		instB := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "b", Namespace: state.Namespace}, instB)).ToNot(HaveOccurred())

		Expect(instA.DeletionTimestamp.IsZero()).To(BeFalse())
		Expect(instB.DeletionTimestamp.IsZero()).To(BeFalse())
	})

	It("should not block deletion if there are no subinstallations left", func() {
		ctx := context.Background()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test1")
		Expect(err).ToNot(HaveOccurred())
		Expect(testenv.InitDefaultContextFromInst(ctx, state, state.Installations[state.Namespace+"/b"]))

		inInstB, err := testutils.CreateInternalInstallation(ctx,
			op.ComponentsRegistry(),
			state.Installations[state.Namespace+"/b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should delete subinstallations if no one imports exported values", func() {
		ctx := context.Background()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test2")
		Expect(err).ToNot(HaveOccurred())
		Expect(testenv.InitDefaultContextFromInst(ctx, state, state.Installations[state.Namespace+"/a"]))

		inInstB, err := testutils.CreateInternalInstallation(ctx,
			op.ComponentsRegistry(),
			state.Installations[state.Namespace+"/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		instC := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
		Expect(instC.DeletionTimestamp.IsZero()).To(BeFalse())
	})

	It("should propagate the force deletion annotation to a execution in deletion state", func() {
		ctx := context.Background()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test3")
		Expect(err).ToNot(HaveOccurred())

		inst := &lsv1alpha1.Installation{}
		inst.Name = "root"
		inst.Namespace = state.Namespace
		Expect(testenv.InitDefaultContextFromInst(ctx, state, inst))
		testutils.ExpectNoError(testenv.Client.Delete(ctx, inst))
		err = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(inst)) // returns a still waiting error
		Expect(err.Error()).To(ContainSubstring("waiting"))

		exec := &lsv1alpha1.Execution{}
		exec.Name = "root"
		exec.Namespace = state.Namespace
		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
		Expect(exec.DeletionTimestamp).ToNot(BeNil())

		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
		metav1.SetMetaDataAnnotation(&inst.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ForceReconcileOperation))
		testutils.ExpectNoError(testenv.Client.Update(ctx, inst))
		_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(inst)) // returns a still waiting error

		// execution should have the force reconcile annotation
		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
		ann, ok := exec.Annotations[lsv1alpha1.OperationAnnotation]
		Expect(ok).To(BeTrue())
		Expect(ann).To(Equal(string(lsv1alpha1.ForceReconcileOperation)))
	})

})
