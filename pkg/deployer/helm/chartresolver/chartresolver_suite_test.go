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
	"testing"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/helm/chartresolver"

	utils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chartresolver Test Suite")
}

var _ = Describe("GetChart", func() {

	Context("FromOCIRegistry", func() {
		It("should resolve a chart from public readable helm ociClient artifact", func() {
			ctx := context.Background()
			defer ctx.Done()
			ociClient, err := ociclient.NewClient(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			chartAccess := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:3.29.0",
			}

			chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})

		It("should resolve a legacy chart from public readable helm ociClient artifact", func() {
			ctx := context.Background()
			defer ctx.Done()
			ociClient, err := ociclient.NewClient(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			chartAccess := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0",
			}

			chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})
	})

	It("should resolve a chart from a public readable component descriptor", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logr.Discard())
		Expect(err).ToNot(HaveOccurred())

		ref := &helmv1alpha1.RemoteChartReference{}
		ref.Reference = &lsv1alpha1.ComponentDescriptorReference{}
		repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("eu.gcr.io/gardener-project/landscaper/tutorials/components", ""))
		Expect(err).ToNot(HaveOccurred())
		ref.Reference.RepositoryContext = &repoCtx
		ref.Reference.ComponentName = "github.com/gardener/landscaper/ingress-nginx"
		ref.Reference.Version = "v0.2.1"
		ref.ResourceName = "ingress-nginx-chart"
		chartAccess := &helmv1alpha1.Chart{
			FromResource: ref,
		}

		chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
		Expect(err).ToNot(HaveOccurred())
		Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
	})

	It("should resolve a chart from an inline component descriptor", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logr.Discard())
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

		chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
		Expect(err).ToNot(HaveOccurred())
		Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
	})

	It("should resolve a chart as base64 encoded file", func() {
		ctx := context.Background()
		ociClient, err := ociclient.NewClient(logr.Discard())
		Expect(err).ToNot(HaveOccurred())

		chartBytes, closer := utils.ReadChartFrom("./testdata/testchart")
		defer closer()

		chartAccess := &helmv1alpha1.Chart{
			Archive: &helmv1alpha1.ArchiveAccess{
				Raw: base64.StdEncoding.EncodeToString(chartBytes),
			},
		}

		chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
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
			ociClient, err := ociclient.NewClient(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart")
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

			chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("testchart"))
		})

		It("should not try to load a chart for non-success http status codes", func() {
			ctx := context.Background()
			defer ctx.Done()
			ociClient, err := ociclient.NewClient(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				body := []byte(http.StatusText(401))
				_, err := w.Write(body)
				Expect(err).ToNot(HaveOccurred())
			}))

			chartAccess := &helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Remote: &helmv1alpha1.RemoteArchiveAccess{
						URL: srv.URL,
					},
				},
			}

			chart, err := chartresolver.GetChart(ctx, logr.Discard(), ociClient, chartAccess)
			Expect(chart).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(http.StatusText(401)))
		})

	})

})
