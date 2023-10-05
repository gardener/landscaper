// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver_test

import (
	"bytes"
	"context"
	"encoding/base64"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
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
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

			chartAccess := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:3.29.0",
			}

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, &lsv1alpha1.Context{UseOCM: false}, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})

		It("should resolve a legacy chart from public readable helm ociClient artifact", func() {
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

			chartAccess := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0",
			}

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, &lsv1alpha1.Context{UseOCM: false}, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})
	})

	It("should resolve a chart as base64 encoded file", func() {
		ctx := logging.NewContext(context.Background(), logging.Discard())

		chartBytes, closer := utils.ReadChartFrom("./testdata/testchart")
		defer closer()

		chartAccess := &helmv1alpha1.Chart{
			Archive: &helmv1alpha1.ArchiveAccess{
				Raw: base64.StdEncoding.EncodeToString(chartBytes),
			},
		}

		chart, err := chartresolver.GetChart(ctx, chartAccess, nil, nil, nil, nil, nil)
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
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

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

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("testchart"))
		})

		It("should not try to load a chart for non-success http status codes", func() {
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

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

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, nil, nil, nil, nil)
			Expect(chart).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(http.StatusText(401)))
		})

	})

})
