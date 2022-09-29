package helmchartrepo

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

var _ = Describe("Catalog cache", func() {

	Context("getCatalogFromCache", func() {

		It("should return a previously cached catalog", func() {
			var (
				repoURL = "test-catalog-url"
				catalog = repo.IndexFile{
					Entries: map[string]repo.ChartVersions{
						"test-chart": {
							&repo.ChartVersion{
								Metadata: &chart.Metadata{Name: "test-chart", Version: "test-version"},
								URLs:     []string{"test-chart-url"},
							},
						},
					},
				}
			)

			Expect(getCatalogCache()).NotTo(BeNil())

			rawCatalog, err := json.Marshal(catalog)
			Expect(err).NotTo(HaveOccurred())

			catalog1, digest1 := getCatalogCache().getCatalogFromCache(repoURL, rawCatalog)
			Expect(catalog1).To(BeNil())
			Expect(digest1).ToNot(BeNil())

			catalog1, err = getCatalogCache().parseCatalog(rawCatalog)
			Expect(err).NotTo(HaveOccurred())
			Expect(*catalog1).To(Equal(catalog))

			getCatalogCache().storeCatalogInCache(repoURL, catalog1, digest1)

			catalog2, digest2 := getCatalogCache().getCatalogFromCache(repoURL, rawCatalog)
			Expect(catalog2).To(Equal(catalog1))
			Expect(digest2).To(Equal(digest1))
		})
	})
})
