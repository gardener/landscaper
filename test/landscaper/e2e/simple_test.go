// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"path/filepath"

	"github.com/gardener/landscaper/pkg/utils"

	clientlib "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	v1alpha1config "github.com/gardener/landscaper/apis/config/v1alpha1"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	execctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	instctlr "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Simple", func() {

	var (
		state *envtest.State

		execActuator, instActuator, mockActuator reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error

		op := operation.NewOperation(api.LandscaperScheme, record.NewFakeRecorder(1024), testenv.Client)
		lsConfig := &v1alpha1config.LandscaperConfiguration{
			Registry: v1alpha1config.RegistryConfiguration{
				Local: &v1alpha1config.LocalRegistryConfiguration{RootPath: filepath.Join(projectRoot, "examples", "01-simple")},
			},
		}
		v1alpha1config.SetDefaults_LandscaperConfiguration(lsConfig)
		lsConfigCore := &config.LandscaperConfiguration{}
		Expect(v1alpha1config.Convert_v1alpha1_LandscaperConfiguration_To_config_LandscaperConfiguration(lsConfig, lsConfigCore, nil)).ToNot(HaveOccurred())
		instActuator = instctlr.NewTestActuator(testenv.Client, testenv.Client, testenv.Client, *op, logging.Discard(),
			clock.RealClock{}, lsConfigCore, "test-inst4-"+testutils.GetNextCounter())

		execActuator, err = execctlr.NewController(testenv.Client, testenv.Client, testenv.Client, testenv.Client, logging.Discard(), api.LandscaperScheme,
			record.NewFakeRecorder(1024), 1000, false, "exec-test-"+testutils.GetNextCounter())
		Expect(err).ToNot(HaveOccurred())

		mockActuator, err = mockctlr.NewController(testenv.Client, testenv.Client, testenv.Client, testenv.Client,
			utils.NewFinishedObjectCache(),
			logging.Discard(), api.LandscaperScheme,
			record.NewFakeRecorder(1024), mockv1alpha1.Configuration{}, "test-simple"+testutils.GetNextCounter())
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
		var err error
		state, err = testenv.InitResources(ctx, filepath.Join(projectRoot, "examples", "01-simple", "cluster"))
		Expect(err).ToNot(HaveOccurred())
		Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

		// first the installation controller should run and set the finalizer
		// afterwards it should again reconcile and deploy the execution
		instReq := testutils.Request("root-1", state.Namespace)
		inst := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).To(Succeed())
		Expect(testutils.AddReconcileAnnotation(ctx, testenv, inst)).To(Succeed())
		testutils.ShouldReconcile(ctx, instActuator, instReq) // add finalizer
		testutils.ShouldReconcile(ctx, instActuator, instReq) // remove reconcile annotation and generate jobID
		testutils.ShouldReconcile(ctx, instActuator, instReq) // create execution; returns error because execution is unfinished

		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).To(Succeed())
		Expect(inst.Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Progressing))
		Expect(inst.Status.JobID).NotTo(BeEmpty())
		Expect(inst.Status.JobIDFinished).To(BeEmpty())
		Expect(inst.Status.ExecutionReference).ToNot(BeNil())

		jobID := inst.Status.JobID

		execReq := testutils.Request(inst.Status.ExecutionReference.Name, inst.Status.ExecutionReference.Namespace)
		exec := &lsv1alpha1.Execution{}
		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(BeEmpty())
		Expect(exec.Status.JobID).To(Equal(jobID))
		Expect(exec.Status.JobIDFinished).To(BeEmpty())

		// reconcile execution
		testutils.ShouldReconcile(ctx, execActuator, execReq) // not finished

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Progressing))
		Expect(exec.Status.JobID).To(Equal(jobID))
		Expect(exec.Status.JobIDFinished).To(BeEmpty())
		deployItems, err := read_write_layer.ListManagedDeployItems(ctx, testenv.Client, clientlib.ObjectKeyFromObject(exec), read_write_layer.R000000)
		Expect(err).To(BeNil())
		Expect(deployItems.Items).To(HaveLen(1))

		di := &deployItems.Items[0]
		Expect(di.Status.Phase).To(BeEmpty())
		Expect(di.Status.GetJobID()).To(Equal(jobID))
		Expect(di.Status.JobIDFinished).To(BeEmpty())

		// reconcile deploy item
		diReq := testutils.Request(di.Name, di.Namespace)
		testutils.ShouldReconcile(ctx, mockActuator, diReq)
		testutils.ShouldReconcile(ctx, mockActuator, diReq)

		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).To(Succeed())
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))
		Expect(di.Status.GetJobID()).To(Equal(jobID))
		Expect(di.Status.JobIDFinished).To(Equal(jobID))

		// as the deploy item is now successfully reconciled, we have to trigger the execution
		// and check if the states are correctly propagated
		testutils.ShouldReconcile(ctx, execActuator, execReq)

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).To(Succeed())
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Succeeded))
		Expect(exec.Status.JobID).To(Equal(jobID))
		Expect(exec.Status.JobIDFinished).To(Equal(jobID))
		Expect(exec.Status.ExportReference).ToNot(BeNil())

		// as the execution is now successfully reconciled, we have to trigger the installation
		// and check if the state is propagated
		testutils.ShouldReconcile(ctx, instActuator, instReq)

		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).To(Succeed())
		Expect(inst.Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Succeeded))
		Expect(inst.Status.JobID).To(Equal(jobID))
		Expect(inst.Status.JobIDFinished).To(Equal(jobID))

		By("delete resource")
		Expect(testenv.Client.Delete(ctx, inst)).To(Succeed())

		// the installation controller should propagate the deletion to the execution
		testutils.ShouldReconcile(ctx, instActuator, instReq)         // generate jobID for deletion
		testutils.ShouldReconcileButRetry(ctx, instActuator, instReq) // delete execution

		Expect(testenv.Client.Get(ctx, instReq.NamespacedName, inst)).To(Succeed())
		Expect(inst.Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Deleting))
		Expect(inst.Status.JobID).NotTo(Equal(jobID))
		Expect(inst.Status.JobIDFinished).To(Equal(jobID))

		deletionJobID := inst.Status.JobID

		Expect(testenv.Client.Get(ctx, execReq.NamespacedName, exec)).To(Succeed())
		Expect(exec.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")
		Expect(exec.Status.ExecutionPhase).To(Equal(lsv1alpha1.ExecutionPhases.Succeeded))
		Expect(exec.Status.JobID).To(Equal(deletionJobID))
		Expect(exec.Status.JobIDFinished).To(Equal(jobID))

		// the execution controller should propagate the deletion to its deploy item
		testutils.ShouldReconcile(ctx, execActuator, execReq)

		Expect(testenv.Client.Get(ctx, diReq.NamespacedName, di)).To(Succeed())
		Expect(di.DeletionTimestamp.IsZero()).To(BeFalse(), "deletion timestamp should be set")
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))
		Expect(di.Status.GetJobID()).To(Equal(deletionJobID))
		Expect(di.Status.JobIDFinished).To(Equal(jobID))

		// deployer should remove finalizer
		testutils.ShouldReconcile(ctx, mockActuator, diReq)
		err = testenv.Client.Get(ctx, diReq.NamespacedName, di)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "deploy item should be deleted")

		// execution controller should remove finalizer
		testutils.ShouldReconcile(ctx, execActuator, execReq)
		err = testenv.Client.Get(ctx, execReq.NamespacedName, exec)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "execution should be deleted")

		// installation controller should remove finalizer
		testutils.ShouldReconcile(ctx, instActuator, instReq)
		err = testenv.Client.Get(ctx, instReq.NamespacedName, inst)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "installation should be deleted")
	})
})
