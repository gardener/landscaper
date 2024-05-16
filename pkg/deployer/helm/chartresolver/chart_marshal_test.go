package chartresolver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/gardener/landscaper/pkg/deployer/helm/chartresolver"
)

var _ = Describe("Chart tree", func() {

	It("should marshal and unmarshal a chart", func() {
		c1, err := loader.LoadDir("./testdata/testchart2")
		Expect(err).NotTo(HaveOccurred())

		raw, err := chartresolver.MarshalChart(c1)
		Expect(err).NotTo(HaveOccurred())

		c2, err := chartresolver.UnmarshalChart(raw)
		Expect(err).NotTo(HaveOccurred())

		checkCharts(c1, c2)
	})

})

func checkCharts(c1, c2 *chart.Chart) {
	Expect(c1.Name()).To(Equal(c2.Name()))
	Expect(len(c1.Templates)).To(Equal(len(c2.Templates)))
	for i := range c1.Templates {
		t1 := c1.Templates[i]
		t2 := c2.Templates[i]
		Expect(t1.Name).To(Equal(t2.Name))
		Expect(t1.Data).To(Equal(t2.Data))
	}
	Expect(len(c1.Dependencies())).To(Equal(len(c2.Dependencies())))
	for i := range c1.Dependencies() {
		d1 := c1.Dependencies()[i]
		d2 := c2.Dependencies()[i]
		checkCharts(d1, d2)
	}
}
