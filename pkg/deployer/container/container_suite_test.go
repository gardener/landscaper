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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/test/utils"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lsutils "github.com/gardener/landscaper/pkg/utils"
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

		_, err = containerctlr.AddControllerToManager(logging.Discard(), mgr, mgr,
			containerv1alpha1.Configuration{}, "template-"+utils.GetNextCounter())
		Expect(err).To(BeNil())

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
})
