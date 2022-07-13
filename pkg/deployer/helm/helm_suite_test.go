// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	lsutils "github.com/gardener/landscaper/pkg/utils"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "helm Test Suite")
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

var _ = Describe("Template", func() {
	It("should ignore non-kubernetes manifests that are valid yaml", func() {
		ctx := context.Background()

		kubeconfig, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		chartData, closer := utils.ReadChartFrom("./testdata/testchart")
		defer closer()
		helmConfig := &helmv1alpha1.ProviderConfiguration{}
		helmConfig.Kubeconfig = base64.StdEncoding.EncodeToString(kubeconfig)
		helmConfig.Chart.Archive = &helmv1alpha1.ArchiveAccess{
			Raw: base64.StdEncoding.EncodeToString(chartData),
		}
		helmConfig.Name = "foo"
		helmConfig.Namespace = "foo"
		providerConfig, err := helper.ProviderConfigurationToRawExtension(helmConfig)
		Expect(err).ToNot(HaveOccurred())

		item := &lsv1alpha1.DeployItem{}
		item.Spec.Configuration = providerConfig

		lsCtx := &lsv1alpha1.Context{}
		lsCtx.Name = lsv1alpha1.DefaultContextName
		lsCtx.Namespace = item.Namespace
		h, err := helm.New(logr.Discard(), helmv1alpha1.Configuration{}, testenv.Client, testenv.Client, item, nil, lsCtx, nil)
		Expect(err).ToNot(HaveOccurred())
		files, crds, _, _, err := h.Template(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(crds).To(HaveKey("testchart/crds/crontabs.yaml"))
		Expect(files).To(HaveKey("testchart/templates/secret.yaml"))
		Expect(files).To(HaveKey("testchart/templates/note.txt"))

		objects, err := kutil.ParseFiles(logr.Discard(), files)
		Expect(err).ToNot(HaveOccurred())
		Expect(objects).To(HaveLen(1))
	})

	Context("Integration", func() {

		var (
			ctx    context.Context
			cancel context.CancelFunc
			state  *envtest.State
			mgr    manager.Manager
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			var err error
			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			mgr, err = manager.New(testenv.Env.Config, manager.Options{
				Scheme:             api.LandscaperScheme,
				MetricsBindAddress: "0",
				NewClient:          lsutils.NewUncachedClient,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(helm.AddDeployerToManager(simplelogger.NewIOLogger(GinkgoWriter), mgr, mgr, helmv1alpha1.Configuration{})).To(Succeed())

			go func() {
				Expect(mgr.Start(ctx)).To(Succeed())
			}()
		})

		AfterEach(func() {
			defer cancel()
			Expect(state.CleanupState(ctx)).To(Succeed())
		})

		It("should create the release namespace if configured", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./chartresolver/testdata/testchart")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       "some-namespace",
				Chart:           chartAccess,
				CreateNamespace: true,
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, kutil.ObjectKey("some-namespace", ""), &corev1.Namespace{}); err != nil {
					return err
				}
				return nil
			}, 10*time.Second, 2*time.Second).Should(Succeed(), "additional namespace should be created")
		})

	})

})
