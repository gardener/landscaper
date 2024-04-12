// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

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
	RunSpecs(t, "Chartresolver Test Suite")
}

var _ = Describe("GetChart", func() {
	var (
		octx ocm.Context
		ctx  context.Context
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	Context("FromOCIRegistry", func() {
		It("should resolve a chart from public readable helm ociClient artifact", func() {
			ref := "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:3.29.0"

			chart, err := getChartFromOCIRef(ctx, nil, &lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: false}}, ref, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})

		It("should resolve a legacy chart from public readable helm ociClient artifact", func() {
			ref := "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0"

			chart, err := getChartFromOCIRef(ctx, nil, &lsv1alpha1.Context{ContextConfiguration: lsv1alpha1.ContextConfiguration{UseOCM: false}}, ref, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("ingress-nginx"))
		})
	})

	It("should resolve a chart as base64 encoded file", func() {
		chartBytes, closer := utils.ReadChartFrom("./testdata/testchart")
		defer closer()

		Archive := &helmv1alpha1.ArchiveAccess{
			Raw: base64.StdEncoding.EncodeToString(chartBytes),
		}

		chart, err := getChartFromArchive(Archive)
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
			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart")
			defer closer()

			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, bytes.NewBuffer(chartBytes))
				Expect(err).ToNot(HaveOccurred())
			}))

			Archive := &helmv1alpha1.ArchiveAccess{
				Remote: &helmv1alpha1.RemoteArchiveAccess{
					URL: srv.URL,
				},
			}

			chart, err := getChartFromArchive(Archive)
			Expect(err).ToNot(HaveOccurred())
			Expect(chart.Metadata.Name).To(Equal("testchart"))
		})

		It("should not try to load a chart for non-success http status codes", func() {
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				body := []byte(http.StatusText(401))
				_, err := w.Write(body)
				Expect(err).ToNot(HaveOccurred())
			}))

			Archive := &helmv1alpha1.ArchiveAccess{
				Remote: &helmv1alpha1.RemoteArchiveAccess{
					URL: srv.URL,
				},
			}

			chart, err := getChartFromArchive(Archive)
			Expect(chart).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(http.StatusText(401)))
		})

	})

	Context("From OCM Resource Ref", func() {
		var (
			resourceRef string
			repoCtx     *cdv2.UnstructuredTypedObject
		)

		BeforeEach(func() {
			// Establish an extra context and ocm context for the setup since the ocm context used here will cache the
			// repository context and component version which would partially defeat the purpose of these tests.
			localCtx := logging.NewContext(context.Background(), logging.Discard())
			localOctx := ocm.New(datacontext.MODE_EXTENDED)
			localCtx = localOctx.BindTo(ctx)

			// Setup Test
			registry, err := registries.GetFactory(true).NewRegistryAccess(localCtx, nil, nil, nil, nil,
				&config.LocalRegistryConfiguration{RootPath: "./testdata/ocmrepo"}, nil, nil)
			Expect(err).To(BeNil())

			cdref := &lsv1alpha1.ComponentDescriptorReference{}
			Expect(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), &cdref)).To(BeNil())
			cv, err := registry.GetComponentVersion(localCtx, cdref)
			Expect(err).To(BeNil())
			Expect(cv).ToNot(BeNil())

			templateFuncs, err := gotemplate.LandscaperTplFuncMap(&blueprints.Blueprint{}, cv, nil, nil)
			Expect(err).To(BeNil())

			getResourceKey := templateFuncs["getResourceKey"].(func(args ...interface{}) (string, error))
			resourceRef, err = getResourceKey(`cd://componentReferences/referenced-landscaper-component/resources/chart`)
			Expect(err).To(BeNil())
			repoCtx = &cdv2.UnstructuredTypedObject{}
			Expect(repoCtx.UnmarshalJSON([]byte(`{"type": "local", "filepath": "./testdata/ocmrepo"}`))).To(BeNil())
		})

		It("should resolve a chart from a local ocm resource", func() {
			chart, err := getChartFromResourceRef(ctx, nil, resourceRef, &lsv1alpha1.Context{
				ContextConfiguration: lsv1alpha1.ContextConfiguration{RepositoryContext: repoCtx},
			}, nil)
			Expect(err).To(BeNil())
			Expect(chart).ToNot(BeNil())
		})
		It("resolve from ocm resource ref with ocm config resolver", func() {
			ocmConfig := &corev1.ConfigMap{
				Data: map[string]string{`.ocmconfig`: `
type: generic.config.ocm.software/v1
configurations:
  - type: ocm.config.ocm.software
    resolvers:
      - repository:
          type: local
          filePath: ./testdata/ocmrepo
        priority: 10
`},
			}
			chart, err := getChartFromResourceRef(ctx, ocmConfig, resourceRef, &lsv1alpha1.Context{}, nil)
			Expect(err).To(BeNil())
			Expect(chart).ToNot(BeNil())
		})
	})
})
