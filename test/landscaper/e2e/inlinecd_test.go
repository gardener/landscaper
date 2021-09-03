// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"path/filepath"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	execctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	instctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Inline Component Descriptor", func() {

	var (
		state                 *envtest.State
		fakeComponentRegistry ctf.ComponentResolver

		execActuator, instActuator, mockActuator reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error
		fakeComponentRegistry, err = componentsregistry.NewLocalClient(logr.Discard(), filepath.Join(projectRoot, "examples", "02-inline-cd"))
		Expect(err).ToNot(HaveOccurred())

		op := operation.NewOperation(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).
			SetComponentsRegistry(fakeComponentRegistry)

		instActuator = instctlr.NewTestActuator(*op, &config.LandscaperConfiguration{
			Registry: config.RegistryConfiguration{
				Local: &config.LocalRegistryConfiguration{RootPath: filepath.Join(projectRoot, "examples", "02-inline-cd")},
			},
		})

		execActuator, err = execctlr.NewController(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024))
		Expect(err).ToNot(HaveOccurred())

		mockActuator, err = mockctlr.NewController(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024), mockv1alpha1.Configuration{})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if state != nil {
			ctx := context.Background()
			defer ctx.Done()
			Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
			state = nil
		}
	})

	It("Should successfully reconcile InlineCDTest", func() {
		ctx := context.Background()

		var err error
		state, err = testenv.InitResources(ctx, filepath.Join(projectRoot, "examples", "02-inline-cd", "cluster"))
		Expect(err).ToNot(HaveOccurred())

		lsCtx := &lsv1alpha1.Context{}
		lsCtx.Name = lsv1alpha1.DefaultContextName
		lsCtx.Namespace = state.Namespace
		lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()
		Expect(state.Create(ctx, lsCtx)).To(Succeed())

		// first the installation controller should run and set the finalizer
		// afterwards it should again reconcile and deploy the execution
		instReq := testutils.Request("root-1", state.Namespace)
		testutils.ShouldReconcile(ctx, instActuator, instReq)
		testutils.ShouldReconcile(ctx, instActuator, instReq)

		inst := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseProgressing))
		Expect(inst.Status.ExecutionReference).ToNot(BeNil())
		Expect(inst.Status.Imports).To(HaveLen(1))
		Expect(inst.Status.Imports[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("imp-a"),
			"Type": Equal(lsv1alpha1.DataImportStatusType),
		}))

		execReq := testutils.Request(inst.Status.ExecutionReference.Name, inst.Status.ExecutionReference.Namespace)
		exec := &lsv1alpha1.Execution{}
		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())

		// after the execution was created by the installation, we need to run the execution controller
		// on first reconcile it should add the finalizer
		// and int he second reconcile it should create the deploy item
		testutils.ShouldReconcile(ctx, execActuator, execReq)
		testutils.ShouldReconcile(ctx, execActuator, execReq)

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))

		diList := &lsv1alpha1.DeployItemList{}
		Expect(testenv.Client.List(ctx, diList)).ToNot(HaveOccurred())
		Expect(diList.Items).To(HaveLen(1))

		diReq := testutils.Request(exec.Status.DeployItemReferences[0].Reference.Name, exec.Status.DeployItemReferences[0].Reference.Namespace)
		di := &lsv1alpha1.DeployItem{}
		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).ToNot(HaveOccurred())

		testutils.ShouldReconcile(ctx, mockActuator, diReq)
		testutils.ShouldReconcile(ctx, mockActuator, diReq)

		// as the deploy item is now successfully reconciled, we have to trigger the execution
		// and check if the states are correctly propagated
		_, err = execActuator.Reconcile(ctx, execReq)
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(exec.Status.ExportReference).ToNot(BeNil())

		// as the execution is now successfully reconciled, we have to trigger the installation
		// and check if the state is propagated
		_, err = instActuator.Reconcile(ctx, testutils.Request("root-1", state.Namespace))
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))

		By("delete resource")
		Expect(testenv.Client.Delete(ctx, inst)).ToNot(HaveOccurred())

		// the installation controller should propagate the deletion to its subcharts
		_, err = instActuator.Reconcile(ctx, instReq)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("waiting for deletion"))

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")

		// the execution controller should propagate the deletion to its deploy item
		_, err = execActuator.Reconcile(ctx, execReq)
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).ToNot(HaveOccurred())
		Expect(di.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")

		_, err = mockActuator.Reconcile(ctx, diReq)
		Expect(err).ToNot(HaveOccurred())
		err = testenv.Client.Get(ctx, diReq.NamespacedName, di)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "deploy item should be deleted")

		// execution controller should remove the finalizer
		testutils.ShouldReconcile(ctx, execActuator, execReq)
		err = testenv.Client.Get(ctx, execReq.NamespacedName, exec)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "execution should be deleted")

		// installation controller should remove its own finalizer
		testutils.ShouldReconcile(ctx, instActuator, instReq)
		err = testenv.Client.Get(ctx, instReq.NamespacedName, inst)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "installation should be deleted")
	})
})
