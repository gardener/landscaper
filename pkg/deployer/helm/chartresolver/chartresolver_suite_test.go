// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	logtesting "github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/helm/chartresolver"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chartresolver Test Suite")
}

var _ = Describe("GetChart", func() {

	It("should resolve a chart from public readable helm oci artifact", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logtesting.NullLogger{})
		Expect(err).ToNot(HaveOccurred())

		chartAccess := &helmv1alpha1.Chart{
			Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0",
		}

		chart, err := chartresolver.GetChart(ctx, logtesting.NullLogger{}, ociClient, chartAccess)
		Expect(err).ToNot(HaveOccurred())
		Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
	})

	It("should resolve a chart from a public readable component descriptor", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logtesting.NullLogger{})
		Expect(err).ToNot(HaveOccurred())

		ref := &helmv1alpha1.RemoteChartReference{}
		ref.Reference = &lsv1alpha1.ComponentDescriptorReference{}
		ref.Reference.RepositoryContext = &cdv2.RepositoryContext{
			Type:    cdv2.OCIRegistryType,
			BaseURL: "eu.gcr.io/gardener-project/landscaper/tutorials/components",
		}
		ref.Reference.ComponentName = "github.com/gardener/landscaper/ingress-nginx"
		ref.Reference.Version = "v0.2.1"
		ref.ResourceName = "ingress-nginx-chart"
		chartAccess := &helmv1alpha1.Chart{
			FromResource: ref,
		}

		chart, err := chartresolver.GetChart(ctx, logtesting.NullLogger{}, ociClient, chartAccess)
		Expect(err).ToNot(HaveOccurred())
		Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
	})

	It("should resolve a chart from an inline component descriptor", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logtesting.NullLogger{})
		Expect(err).ToNot(HaveOccurred())

		file, err := ioutil.ReadFile("./testdata/01-component-descriptor.yaml")
		Expect(err).ToNot(HaveOccurred())

		inline := &cdv2.ComponentDescriptor{}
		err = yaml.Unmarshal(file, &inline)
		Expect(err).ToNot(HaveOccurred())

		ref := &helmv1alpha1.RemoteChartReference{}
		ref.Inline = inline
		ref.ResourceName = "ingress-nginx-chart"

		chartAccess := &helmv1alpha1.Chart{
			FromResource: ref,
		}

		chart, err := chartresolver.GetChart(ctx, logtesting.NullLogger{}, ociClient, chartAccess)
		Expect(err).ToNot(HaveOccurred())
		Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
	})

	It("should resolve a chart as base64 encoded file", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logtesting.NullLogger{})
		Expect(err).ToNot(HaveOccurred())

		chartBytes, closer := readChartFrom("./testdata/testchart")
		defer closer()

		chartAccess := &helmv1alpha1.Chart{
			Archive: &helmv1alpha1.ArchiveAccess{
				Raw: base64.StdEncoding.EncodeToString(chartBytes),
			},
		}

		chart, err := chartresolver.GetChart(ctx, logtesting.NullLogger{}, ociClient, chartAccess)
		Expect(err).ToNot(HaveOccurred())
		Expect(chart.Metadata.Name).To(Equal("testchart"))
	})

	Context("remote url", func() {

		var (
			srv *httptest.Server
		)

		AfterEach(func() {
			if srv == nil {
				return
			}
			srv.Close()
		})

		It("should resolve a chart from a webserver", func() {
			ctx := context.Background()
			defer ctx.Done()
			ociClient, err := ociclient.NewClient(logtesting.NullLogger{})
			Expect(err).ToNot(HaveOccurred())

			chartBytes, closer := readChartFrom("./testdata/testchart")
			defer closer()

			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, bytes.NewBuffer(chartBytes))
				Expect(err).ToNot(HaveOccurred())
			}))

			chartAccess := &helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Remote: &helmv1alpha1.RemoteArchiveAccess{
						URL: srv.URL,
					},
				},
			}

			chart, err := chartresolver.GetChart(ctx, logtesting.NullLogger{}, ociClient, chartAccess)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("testchart"))
		})

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
