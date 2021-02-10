// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	logtesting "github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
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

var _ = Describe("RunInstallation", func() {
	It("should ignore non-kubernetes manifests that are valid yaml", func() {
		ctx := context.Background()
		defer ctx.Done()

		kubeconfig, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		chartData, closer := readChartFrom("./testdata/testchart")
		defer closer()
		helmConfig := &helmv1alpha1.ProviderConfiguration{}
		helmConfig.Kubeconfig = base64.StdEncoding.EncodeToString(kubeconfig)
		helmConfig.Chart.Archive = &helmv1alpha1.ArchiveAccess{
			Raw: base64.StdEncoding.EncodeToString(chartData),
		}
		providerConfig, err := helper.ProviderConfigurationToRawExtension(helmConfig)
		Expect(err).ToNot(HaveOccurred())

		item := &lsv1alpha1.DeployItem{}
		item.Spec.Configuration = providerConfig

		h, err := helm.New(logtesting.NullLogger{}, &helmv1alpha1.Configuration{}, testenv.Client, item, nil)
		Expect(err).ToNot(HaveOccurred())
		files, _, err := h.Template(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(files).To(HaveKey("testchart/templates/secret.yaml"))
		Expect(files).To(HaveKey("testchart/templates/note.txt"))

		objects, err := kutil.ParseFiles(logtesting.NullLogger{}, files)
		Expect(err).ToNot(HaveOccurred())
		Expect(objects).To(HaveLen(1))
	})
})

func readChartFrom(path string) ([]byte, func()) {
	chart, err := chartloader.LoadDir(path)
	Expect(err).ToNot(HaveOccurred())
	tempDir, err := ioutil.TempDir(os.TempDir(), "chart-")
	Expect(err).ToNot(HaveOccurred())
	closer := func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	}

	chartPath, err := chartutil.Save(chart, tempDir)
	Expect(err).ToNot(HaveOccurred())

	chartBytes, err := ioutil.ReadFile(chartPath)
	Expect(err).ToNot(HaveOccurred())
	return chartBytes, closer
}
