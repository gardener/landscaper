// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"path/filepath"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	"github.com/gardener/landscaper/pkg/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	"github.com/gardener/landscaper/pkg/kubernetes"
	execctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	instctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Simple", func() {

	var (
		state                 *envtest.State
		fakeComponentRegistry ctf.ComponentResolver

		execActuator, instActuator, mockActuator reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error
		fakeComponentRegistry, err = componentsregistry.NewLocalClient(testing.NullLogger{}, filepath.Join(projectRoot, "examples", "01-simple"))
		Expect(err).ToNot(HaveOccurred())

		op := operation.NewOperation(log.NullLogger{}, testenv.Client, kubernetes.LandscaperScheme, fakeComponentRegistry)

		instActuator = instctlr.NewTestActuator(op, &config.LandscaperConfiguration{
			Registry: config.RegistryConfiguration{
				Local: &config.LocalRegistryConfiguration{RootPath: filepath.Join(projectRoot, "examples", "01-simple")},
			},
		})

		execActuator, err = execctlr.NewActuator()
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.ClientInto(testenv.Client, execActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.SchemeInto(kubernetes.LandscaperScheme, execActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.LoggerInto(testing.NullLogger{}, execActuator)
		Expect(err).ToNot(HaveOccurred())

		mockActuator, err = mockctlr.NewActuator()
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.ClientInto(testenv.Client, mockActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.SchemeInto(kubernetes.LandscaperScheme, mockActuator)
		Expect(err).ToNot(HaveOccurred())
		_, err = inject.LoggerInto(testing.NullLogger{}, mockActuator)
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

	It("Should successfully reconcile SimpleTest", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, filepath.Join(projectRoot, "examples", "01-simple", "cluster"))
		Expect(err).ToNot(HaveOccurred())

		// first the installation controller should run and set the finalizer
		// afterwards it should again reconcile and deploy the execution
		instReq := request("root-1", state.Namespace)
		testutils.ShouldReconcile(instActuator, instReq)
		testutils.ShouldReconcile(instActuator, instReq)

		inst := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseProgressing))
		Expect(inst.Status.ExecutionReference).ToNot(BeNil())
		Expect(inst.Status.Imports).To(HaveLen(1))
		Expect(inst.Status.Imports[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("imp-a"),
			"Type": Equal(lsv1alpha1.DataImportStatusType),
		}))

		execReq := request(inst.Status.ExecutionReference.Name, inst.Status.ExecutionReference.Namespace)
		exec := &lsv1alpha1.Execution{}
		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())

		// after the execution was created by the installation, we need to run the execution controller
		// on first reconcile it should add the finalizer
		// and int he second reconcile it should create the deploy item
		testutils.ShouldReconcile(execActuator, execReq)
		testutils.ShouldReconcile(execActuator, execReq)

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
		Expect(exec.Status.DeployItemReferences).To(HaveLen(1))

		diList := &lsv1alpha1.DeployItemList{}
		Expect(testenv.Client.List(ctx, diList)).ToNot(HaveOccurred())
		Expect(diList.Items).To(HaveLen(1))

		diReq := request(exec.Status.DeployItemReferences[0].Reference.Name, exec.Status.DeployItemReferences[0].Reference.Namespace)
		di := &lsv1alpha1.DeployItem{}
		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).ToNot(HaveOccurred())

		testutils.ShouldReconcile(mockActuator, diReq)
		testutils.ShouldReconcile(mockActuator, diReq)

		// as the deploy item is now successfully reconciled, we have to trigger the execution
		// and check if the states are correctly propagated
		_, err = execActuator.Reconcile(execReq)
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(exec.Status.ExportReference).ToNot(BeNil())

		// as the execution is now successfully reconciled, we have to trigger the installation
		// and check if the state is propagated
		_, err = instActuator.Reconcile(request("root-1", state.Namespace))
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).ToNot(HaveOccurred())
		Expect(inst.Status.Phase).To(Equal(lsv1alpha1.ComponentPhaseSucceeded))

		By("delete resource")
		Expect(testenv.Client.Delete(ctx, inst)).ToNot(HaveOccurred())

		// the installation controller should propagate the deletion to its subcharts
		_, err = instActuator.Reconcile(instReq)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("waiting for deletion"))

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).ToNot(HaveOccurred())
		Expect(exec.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")

		// the execution controller should propagate the deletion to its deploy item
		_, err = execActuator.Reconcile(execReq)
		Expect(err).ToNot(HaveOccurred())

		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).ToNot(HaveOccurred())
		Expect(di.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")

		_, err = mockActuator.Reconcile(diReq)
		Expect(err).ToNot(HaveOccurred())
		err = testenv.Client.Get(ctx, diReq.NamespacedName, di)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "deploy item should be deleted")

		// execution controller should remove the finalizer
		testutils.ShouldReconcile(execActuator, execReq)
		err = testenv.Client.Get(ctx, execReq.NamespacedName, exec)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "execution should be deleted")

		// installation controller should remove its own finalizer
		testutils.ShouldReconcile(instActuator, instReq)
		err = testenv.Client.Get(ctx, instReq.NamespacedName, inst)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "installation should be deleted")
	})
})

func request(name, namespace string) reconcile.Request {
	req := reconcile.Request{}
	req.Name = name
	req.Namespace = namespace
	return req
}
