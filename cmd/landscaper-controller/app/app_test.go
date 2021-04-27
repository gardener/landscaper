// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	lsinstall "github.com/gardener/landscaper/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Controller Command Test Suite")
}

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("Landscaper Controller", func() {

	Context("Options", func() {

		It("should parse enabled deployers", func() {
			opts := NewOptions()
			opts.deployer.deployers = "deployer1,deployer2"
			Expect(opts.Complete()).To(Succeed())

			Expect(opts.deployer.EnabledDeployers).To(ConsistOf("deployer1", "deployer2"))
		})

	})

	Context("Deployer Bootstrap", func() {

		var (
			mgr   manager.Manager
			state *envtest.State
		)

		BeforeEach(func() {
			var (
				ctx = context.Background()
				err error
			)
			defer ctx.Done()
			mgr, err = manager.New(testenv.Env.Config, manager.Options{
				MetricsBindAddress: "0",
			})
			Expect(err).ToNot(HaveOccurred())
			lsinstall.Install(mgr.GetScheme())

			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			ctx := context.Background()
			defer ctx.Done()
			Expect(state.CleanupState(ctx, testenv.Client, nil)).To(Succeed())
		})

		It("should create a container deployer registration", func() {
			ctx := context.Background()
			defer ctx.Done()
			opts := NewOptions()
			opts.log = logr.Discard()
			opts.deployer.EnabledDeployers = []string{"container"}
			opts.config = &config.LandscaperConfiguration{}
			opts.config.DeployerManagement.Namespace = state.Namespace
			opts.config.DeployerManagement.Agent.Namespace = state.Namespace

			Expect(opts.deployInternalDeployers(ctx, mgr)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("container", ""), reg)).To(Succeed())
			Expect(reg.Spec.DeployItemTypes).To(ConsistOf(containerctlr.Type))
			Expect(reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName).To(Equal("github.com/gardener/landscaper/container-deployer"))
			Expect(reg.Spec.InstallationTemplate.Blueprint.Reference.ResourceName).To(Equal("container-deployer-blueprint"))
		})

		It("should deploy a helm deployer registration", func() {
			ctx := context.Background()
			defer ctx.Done()
			opts := NewOptions()
			opts.log = logr.Discard()
			opts.deployer.EnabledDeployers = []string{"helm"}
			opts.config = &config.LandscaperConfiguration{}
			opts.config.DeployerManagement.Namespace = state.Namespace
			opts.config.DeployerManagement.Agent.Namespace = state.Namespace

			Expect(opts.deployInternalDeployers(ctx, mgr)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("helm", ""), reg)).To(Succeed())
			Expect(reg.Spec.DeployItemTypes).To(ConsistOf(helmctlr.Type))
			Expect(reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName).To(Equal("github.com/gardener/landscaper/helm-deployer"))
			Expect(reg.Spec.InstallationTemplate.Blueprint.Reference.ResourceName).To(Equal("helm-deployer-blueprint"))
			Expect(reg.Spec.InstallationTemplate.ImportDataMappings).To(HaveKey("targetSelectors"))

			targetSelectorBytes := reg.Spec.InstallationTemplate.ImportDataMappings["targetSelectors"]
			var targetSelector []lsv1alpha1.TargetSelector
			Expect(json.Unmarshal(targetSelectorBytes.RawMessage, &targetSelector)).To(Succeed())
			Expect(targetSelector).To(HaveLen(1))
		})

	})

})
