// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gardener/landscaper/pkg/utils/resource_cache"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/helm/chartresolver"
	utils "github.com/gardener/landscaper/test/utils"
)

var (
	componentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component",
  "version": "1.0.0"
}
`
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chart resolver Test Suite")
}

var _ = Describe("GetChart", func() {

	Context("FromOCIRegistry", func() {
		It("should resolve a chart from public readable helm ociClient artifact", func() {
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

			chartAccess := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:3.29.0",
			}

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: false}},
				nil, nil, nil, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})

		It("should resolve a legacy chart from public readable helm ociClient artifact", func() {
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

			chartAccess := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0",
			}

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: false}},
				nil, nil, nil, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})

		It("should test chart cache 1", func() {

			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

			helmChartCache := chartresolver.GetHelmChartCache(resource_cache.MaxSizeInByteDefault,
				resource_cache.RemoveOutdatedDurationDefault)

			helmChartCache.Clear()
			cacheEntries, size, _ := helmChartCache.GetEntries()
			Expect(len(cacheEntries)).To(Equal(0))
			Expect(size).To(Equal(int64(0)))

			timeBefore := time.Now()

			// fetch a 1. time
			chartAccess1 := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0",
			}

			chart1, err := chartresolver.GetChart(ctx, chartAccess1, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)
			Expect(err).ToNot(HaveOccurred())

			cacheEntries1, size1, _ := helmChartCache.GetEntries()
			Expect(len(cacheEntries1)).To(Equal(1))
			Expect(size1 > 1000).To(BeTrue())
			for _, entry := range cacheEntries1 {
				_, timeTmp := entry.GetEntries()
				Expect(timeBefore.After(timeTmp)).To(BeFalse())
			}

			// fetch a 2. time
			timeBefore = time.Now()
			time.Sleep(time.Duration(10) * time.Millisecond)

			chart2, err := chartresolver.GetChart(ctx, chartAccess1, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)
			Expect(err).ToNot(HaveOccurred())

			chart1.Raw = nil
			Expect(reflect.DeepEqual(chart1, chart2)).To(BeTrue())

			cacheEntries2, size2, _ := helmChartCache.GetEntries()
			Expect(len(cacheEntries2)).To(Equal(1))
			Expect(size2 > 1000).To(BeTrue())

			for _, entry := range cacheEntries1 {
				_, timeTmp := entry.GetEntries()
				Expect(timeBefore.Before(timeTmp)).To(BeTrue())
			}

			contained, err := helmChartCache.HasKey(chartAccess1.Ref, chartAccess1.HelmChartRepo, chartAccess1.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeTrue())

			// fetch a 3. time
			time.Sleep(time.Duration(10) * time.Millisecond)

			chart3, err := chartresolver.GetChart(ctx, chartAccess1, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(chart2, chart3)).To(BeTrue())
			cacheEntries3, size3, _ := helmChartCache.GetEntries()
			Expect(len(cacheEntries3)).To(Equal(1))
			Expect(size2 == size3).To(BeTrue())

			// 4. fetch a new one
			time.Sleep(time.Duration(10) * time.Millisecond)

			chartAccess4 := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17",
			}

			chart4, err := chartresolver.GetChart(ctx, chartAccess4, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart4).ToNot(BeNil())

			cacheEntries4, size4, _ := helmChartCache.GetEntries()
			Expect(len(cacheEntries4)).To(Equal(2))
			Expect(size4 > size3).To(BeTrue())

			contained, err = helmChartCache.HasKey(chartAccess1.Ref, chartAccess1.HelmChartRepo, chartAccess1.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeTrue())

			contained, err = helmChartCache.HasKey(chartAccess4.Ref, chartAccess4.HelmChartRepo, chartAccess4.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeTrue())

			// 5. fetch another one removing the first
			time.Sleep(time.Duration(10) * time.Millisecond)

			newMaxSize := int64(100000)
			helmChartCache.SetMaxSizeInByte(newMaxSize)
			chartAccess5 := &helmv1alpha1.Chart{
				Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.18",
			}

			_, err = chartresolver.GetChart(ctx, chartAccess5, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)
			Expect(err).ToNot(HaveOccurred())
			cacheEntries5, size5, _ := helmChartCache.GetEntries()
			Expect(len(cacheEntries5)).To(Equal(2))
			Expect(size5 <= newMaxSize).To(BeTrue())

			contained, err = helmChartCache.HasKey(chartAccess1.Ref, chartAccess1.HelmChartRepo, chartAccess1.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeFalse())

			contained, err = helmChartCache.HasKey(chartAccess4.Ref, chartAccess4.HelmChartRepo, chartAccess4.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeTrue())

			contained, err = helmChartCache.HasKey(chartAccess5.Ref, chartAccess5.HelmChartRepo, chartAccess5.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeTrue())

			// 6 remove outdated
			time.Sleep(time.Duration(1) * time.Second)
			timeBefore = time.Now()
			helmChartCache.SetMaxSizeInByte(resource_cache.MaxSizeInByteDefault)

			_, _ = chartresolver.GetChart(ctx, chartAccess1, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)

			outdatedDuration := time.Since(timeBefore) + time.Duration(500)*time.Millisecond
			helmChartCache.SetOutdatedDuration(outdatedDuration)
			helmChartCache.SetLastCleanup(time.Now().Add(-(time.Duration(61) * time.Minute)))

			_, _ = chartresolver.GetChart(ctx, chartAccess1, nil,
				&lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: true}},
				nil, nil, nil, true)

			contained, err = helmChartCache.HasKey(chartAccess1.Ref, chartAccess1.HelmChartRepo, chartAccess1.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeTrue())

			contained, err = helmChartCache.HasKey(chartAccess4.Ref, chartAccess4.HelmChartRepo, chartAccess4.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeFalse())

			contained, err = helmChartCache.HasKey(chartAccess5.Ref, chartAccess5.HelmChartRepo, chartAccess5.ResourceRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(contained).To(BeFalse())
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

		chart, err := chartresolver.GetChart(ctx, chartAccess, nil, nil, nil, nil, nil, false)
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

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, nil, nil, nil, nil, false)
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

			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, nil, nil, nil, nil, false)
			Expect(chart).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(http.StatusText(401)))
		})

	})

	Context("From OCM Resource Ref", func() {
		It("should resolve a chart from a local ocm resource", func() {
			ctx := logging.NewContext(context.Background(), logging.Discard())
			defer ctx.Done()

			// Setup Test
			registry, err := registries.GetFactory(true).NewRegistryAccess(ctx, nil, nil, nil,
				&config.LocalRegistryConfiguration{RootPath: "./testdata/ocmrepo"}, nil, nil)
			Expect(err).To(BeNil())

			cdref := &lsv1alpha1.ComponentDescriptorReference{}
			Expect(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), &cdref)).To(BeNil())
			cv, err := registry.GetComponentVersion(ctx, cdref)
			Expect(err).To(BeNil())
			Expect(cv).ToNot(BeNil())

			templateFuncs, err := gotemplate.LandscaperTplFuncMap(&blueprints.Blueprint{}, cv, nil, nil)
			Expect(err).To(BeNil())

			getResourceKey := templateFuncs["getResourceKey"].(func(args ...interface{}) (string, error))
			resourceRef, err := getResourceKey(`cd://componentReferences/referenced-landscaper-component/resources/chart`)
			Expect(err).To(BeNil())
			chartAccess := &helmv1alpha1.Chart{
				ResourceRef: resourceRef,
			}
			un := &cdv2.UnstructuredTypedObject{}
			Expect(un.UnmarshalJSON([]byte(`{"type": "local", "filepath": "./testdata/ocmrepo"}`))).To(BeNil())

			helmChartCache := chartresolver.GetHelmChartCache(resource_cache.MaxSizeInByteDefault,
				resource_cache.RemoveOutdatedDurationDefault)

			helmChartCache.Clear()

			// Test
			chart, err := chartresolver.GetChart(ctx, chartAccess, nil, &lsv1alpha1.Context{
				ContextConfiguration: lsv1alpha1.ContextConfiguration{RepositoryContext: un},
			}, nil, nil, nil, true)
			Expect(err).To(BeNil())
			Expect(chart).ToNot(BeNil())

			contained, err := helmChartCache.HasKey(chartAccess.Ref, chartAccess.HelmChartRepo, chartAccess.ResourceRef)
			Expect(err).To(BeNil())
			Expect(contained).To(BeTrue())
		})
	})

})
