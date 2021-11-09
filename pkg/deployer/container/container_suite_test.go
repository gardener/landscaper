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

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/test/utils"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
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
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(containerctlr.AddControllerToManager(logr.Discard(), mgr, mgr, containerv1alpha1.Configuration{})).To(Succeed())

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
		item, err := containerctlr.NewDeployItemBuilder().ProviderConfig(&containerv1alpha1.ProviderConfiguration{
			RegistryPullSecrets: []lsv1alpha1.ObjectReference{
				{},
			},
		}).Build()
		Expect(err).ToNot(HaveOccurred())
		item.Name = "container-test"
		item.Namespace = state.Namespace

		Expect(state.Create(ctx, item)).To(Succeed())

		di := &lsv1alpha1.DeployItem{}
		Eventually(func() error {
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(item), di)).To(Succeed())
			if di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
				return nil
			}
			return fmt.Errorf("phase is %s but expected it to be failed", di.Status.Phase)
		}, 10*time.Second, 2*time.Second).Should(Succeed())
		Expect(di.Status.LastError).ToNot(BeNil())
		Expect(di.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorConfigurationProblem))
	})
})
