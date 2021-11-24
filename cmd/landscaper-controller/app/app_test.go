// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/cmd/landscaper-controller/app"
	deployerconfig "github.com/gardener/landscaper/pkg/deployermanagement/config"

	"github.com/gardener/landscaper/apis/config"
	lsinstall "github.com/gardener/landscaper/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
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
			Expect(state.CleanupState(ctx)).To(Succeed())
			// remove all Deployer registrations
			deployerRegistrations := &lsv1alpha1.DeployerRegistrationList{}
			Expect(testenv.Client.List(ctx, deployerRegistrations)).To(Succeed())
			for _, reg := range deployerRegistrations.Items {
				d := 10 * time.Second
				Expect(envtest.CleanupForObject(ctx, testenv.Client, &reg, d)).To(Succeed())
			}
		})

		It("should create a container Deployer registration", func() {
			ctx := context.Background()
			defer ctx.Done()
			opts := app.NewOptions()
			opts.Log = logr.Discard()
			opts.Deployer.EnabledDeployers = []string{"container"}
			opts.Config = &config.LandscaperConfiguration{}
			opts.Config.DeployerManagement.Namespace = state.Namespace
			opts.Config.DeployerManagement.Agent.Namespace = state.Namespace

			Expect(opts.DeployInternalDeployers(ctx, mgr)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("container", ""), reg)).To(Succeed())
			Expect(reg.Spec.DeployItemTypes).To(ConsistOf(lsv1alpha1.DeployItemType(deployerconfig.ContainerDeployerType)))
			Expect(reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName).To(Equal("github.com/gardener/landscaper/container-deployer"))
			Expect(reg.Spec.InstallationTemplate.Blueprint.Reference.ResourceName).To(Equal("container-deployer-blueprint"))
		})

		It("should deploy a helm Deployer registration", func() {
			ctx := context.Background()
			defer ctx.Done()
			opts := app.NewOptions()
			opts.Log = logr.Discard()
			opts.Deployer.EnabledDeployers = []string{"helm"}
			opts.Config = &config.LandscaperConfiguration{}
			opts.Config.DeployerManagement.Namespace = state.Namespace
			opts.Config.DeployerManagement.Agent.Namespace = state.Namespace

			Expect(opts.DeployInternalDeployers(ctx, mgr)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("helm", ""), reg)).To(Succeed())
			Expect(reg.Spec.DeployItemTypes).To(ConsistOf(lsv1alpha1.DeployItemType(deployerconfig.HelmDeployerType)))
			Expect(reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName).To(Equal("github.com/gardener/landscaper/helm-deployer"))
			Expect(reg.Spec.InstallationTemplate.Blueprint.Reference.ResourceName).To(Equal("helm-deployer-blueprint"))
		})

		It("should deploy a mock Deployer registration with custom values", func() {
			ctx := context.Background()
			defer ctx.Done()
			opts := app.NewOptions()
			opts.Log = logr.Discard()
			opts.Deployer.EnabledDeployers = []string{"mock"}
			opts.Config = &config.LandscaperConfiguration{}
			opts.Config.DeployerManagement.Namespace = state.Namespace
			opts.Config.DeployerManagement.Agent.Namespace = state.Namespace
			opts.Deployer.DeployersConfig = deployerconfig.DeployersConfiguration{
				Deployers: map[string]deployerconfig.DeployerConfiguration{
					"mock": {
						Type: deployerconfig.ValuesType,
						Values: map[string]interface{}{
							"somekey": "someval",
						},
					},
				},
			}

			Expect(opts.DeployInternalDeployers(ctx, mgr)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("mock", ""), reg)).To(Succeed())
			Expect(reg.Spec.DeployItemTypes).To(ConsistOf(lsv1alpha1.DeployItemType(deployerconfig.MockDeployerType)))
			Expect(reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName).To(Equal("github.com/gardener/landscaper/mock-deployer"))
			Expect(reg.Spec.InstallationTemplate.Blueprint.Reference.ResourceName).To(Equal("mock-deployer-blueprint"))
			Expect(reg.Spec.InstallationTemplate.ImportDataMappings).To(HaveKey("values"))

			Expect(reg.Spec.InstallationTemplate.ImportDataMappings["values"].RawMessage).To(MatchYAML(`
somekey: someval
`))
		})

		It("should deploy a mock Deployer registration with a custom Deployer registration", func() {
			ctx := context.Background()
			defer ctx.Done()
			reg := &lsv1alpha1.DeployerRegistration{}
			opts := app.NewOptions()
			opts.Log = logr.Discard()
			opts.Deployer.EnabledDeployers = []string{"mock"}
			opts.Config = &config.LandscaperConfiguration{}
			opts.Config.DeployerManagement.Namespace = state.Namespace
			opts.Config.DeployerManagement.Agent.Namespace = state.Namespace
			opts.Deployer.DeployersConfig = deployerconfig.DeployersConfiguration{
				Deployers: map[string]deployerconfig.DeployerConfiguration{
					"mock": {
						Type:                 deployerconfig.DeployerRegistrationType,
						DeployerRegistration: reg,
					},
				},
			}

			reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{"test"}
			reg.Spec.InstallationTemplate.Blueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{ResourceName: "some-blueprint"}
			reg.Spec.InstallationTemplate.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{
				"values": lsv1alpha1.NewAnyJSON([]byte("{\"somekey\": \"someval\"}")),
				"other":  lsv1alpha1.NewAnyJSON([]byte("true")),
			}

			Expect(opts.DeployInternalDeployers(ctx, mgr)).To(Succeed())

			reg = &lsv1alpha1.DeployerRegistration{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("mock", ""), reg)).To(Succeed())
			Expect(reg.Spec.DeployItemTypes).To(ConsistOf(lsv1alpha1.DeployItemType("test")))
			Expect(reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName).To(Equal("github.com/gardener/landscaper/mock-deployer"))
			Expect(reg.Spec.InstallationTemplate.Blueprint.Reference.ResourceName).To(Equal("some-blueprint"))
			Expect(reg.Spec.InstallationTemplate.ImportDataMappings).To(HaveKey("values"))

			Expect(reg.Spec.InstallationTemplate.ImportDataMappings["values"].RawMessage).To(MatchYAML(`
somekey: someval
`))
			Expect(reg.Spec.InstallationTemplate.ImportDataMappings["other"].RawMessage).To(MatchYAML("true"))
		})

	})

})
