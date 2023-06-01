package ocmfacade

import (
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/runtime"

	. "github.com/open-component-model/ocm/pkg/testutils"
)

var data = `{"repositoryContext":{"baseUrl":"eu.gcr.io/gardener-project/landscaper/examples","type":"ociRegistry"},"componentName":"github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource","version":"1.0.0"}`

var _ = Describe("ocm-lib facade implementation", func() {
	It("get component version from component descriptor reference", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(data), &cdref))

		r := &RegistryAccess{}
		cv := Must(r.GetComponentVersion(nil, cdref))
		Expect(cv).NotTo(BeNil())
		Expect(cv.GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(cv.GetVersion()).To(Equal("1.0.0"))
	})
	It("test component version methods", func() {

	})

})
