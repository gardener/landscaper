package ocmfacade

import (
	"encoding/json"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var data = `{"repositoryContext":{"baseUrl":"eu.gcr.io/gardener-project/landscaper/examples","type":"ociRegistry"},"componentName":"github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource","version":"1.0.0"}`

var _ = Describe("ocm lib implementation", func() {
	It("get ocm repository and thereby component version from ComponentDescriptorReference", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		err := json.Unmarshal([]byte(data), &cdref)
		Expect(err).To(BeNil())

		r := &RegistryAccess{}
		cv, err := r.GetComponentVersion(nil, cdref)
		Expect(err).To(BeNil())
		Expect(cv).NotTo(BeNil())

		Expect(cv.GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(cv.GetVersion()).To(Equal("1.0.0"))
		cd := cv.GetComponentDescriptor()
		Expect(cd).NotTo(BeNil())

	})

})
