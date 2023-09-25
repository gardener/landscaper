// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "container deployer Test Suite")
}

var (
	testenv     *envtest.Environment
	hostTestEnv *envtest.Environment
	projectRoot = filepath.Join("../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())

	hostTestEnv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = hostTestEnv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
	Expect(hostTestEnv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("Template", func() {

	var (
		state  *envtest.State
		mgr    manager.Manager
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		mgr, err = manager.New(testenv.Env.Config, manager.Options{
			Scheme:             api.LandscaperScheme,
			MetricsBindAddress: "0",
			NewClient:          lsutils.NewUncachedClient(lsutils.LsResourceClientBurstDefault, lsutils.LsResourceClientQpsDefault),
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(containerctlr.AddControllerToManager(logging.Discard(), mgr, mgr,
			containerv1alpha1.Configuration{}, "template-"+utils.GetNextCounter())).To(Succeed())

		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

		go func() {
			Expect(mgr.Start(ctx)).To(Succeed())
		}()
		Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
	})

	AfterEach(func() {
		cancel()
	})

	It("should set phase to failed if the provider configuration is invalid", func() {
		// create a deploy item that is invalid since name and namespace of the registry pull secret are missing
		item, err := containerctlr.NewDeployItemBuilder().
			Key(state.Namespace, "container-test").
			ProviderConfig(&containerv1alpha1.ProviderConfiguration{
				RegistryPullSecrets: []lsv1alpha1.ObjectReference{
					{},
				},
			}).
			GenerateJobID().
			Build()
		Expect(err).ToNot(HaveOccurred())

		Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

		di := &lsv1alpha1.DeployItem{}
		Eventually(func() error {
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(item), di)).To(Succeed())
			if di.Status.Phase.IsFailed() &&
				di.Status.GetJobID() == di.Status.JobIDFinished {
				return nil
			}
			return fmt.Errorf("phase is %s but expected it to be failed", di.Status.Phase)
		}, 10*time.Second, 1*time.Second).Should(Succeed())
		Expect(di.Status.GetLastError()).ToNot(BeNil())
		Expect(di.Status.GetLastError().Codes).To(ContainElement(lsv1alpha1.ErrorConfigurationProblem))
	})

	It("should time out at checkpoints of the container deployer", func() {
		// This test creates/deletes a container deploy item. Before these operations,
		// it replaces the standard timeout checker by test implementations that throw a timeout error during
		// these operations. It verifies that the expected timeouts actually occur.

		isFinished := func(item *lsv1alpha1.DeployItem) bool {
			if err := state.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item); err != nil {
				return false
			}
			return item.Status.JobIDFinished == item.Status.JobID
		}

		item, err := containerctlr.NewDeployItemBuilder().
			Key(state.Namespace, "container-timeout-test").
			ProviderConfig(&containerv1alpha1.ProviderConfiguration{}).
			GenerateJobID().
			Build()
		Expect(err).ToNot(HaveOccurred())

		checkpoint := containerctlr.TimeoutCheckpointContainerStartReconcile
		timeout.ActivateCheckpointTimeoutChecker(checkpoint)
		defer timeout.ActivateStandardTimeoutChecker()

		Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

		Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue())
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item)).To(Succeed())
		Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed))
		Expect(item.Status.LastError).NotTo(BeNil())
		Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
		Expect(item.Status.LastError.Message).To(ContainSubstring(checkpoint))

		Expect(state.Client.Delete(ctx, item)).To(Succeed())
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item)).To(Succeed())

		checkpoint = containerctlr.TimeoutCheckpointContainerStartDelete
		timeout.ActivateCheckpointTimeoutChecker(checkpoint)
		item.Status.SetJobID(uuid.New().String())
		Expect(state.Client.Status().Update(ctx, item)).To(Succeed())

		Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue())
		Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item)).To(Succeed())
		Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.DeleteFailed))
		Expect(item.Status.LastError).NotTo(BeNil())
		Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
		Expect(item.Status.LastError.Message).To(ContainSubstring(checkpoint))
	})

})
