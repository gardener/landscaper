package ocmlib

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/apis/core/v1alpha1"

	. "github.com/open-component-model/ocm/pkg/testutils"
)

var data = `{"repositoryContext":{"baseUrl":"eu.gcr.io/gardener-project/landscaper/examples","type":"ociRegistry"},"componentName":"github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource","version":"1.0.0"}`

var _ = Describe("ocm-lib facade implementation", func() {
	It("get component version from component descriptor reference", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(data), &cdref))

		r := &RegistryAccess{
			octx:    ocm.DefaultContext(),
			session: ocm.NewSession(datacontext.NewSession()),
		}

		cv := Must(r.GetComponentVersion(context.Background(), cdref))
		Expect(cv).NotTo(BeNil())
		Expect(cv.GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(cv.GetVersion()).To(Equal("1.0.0"))
	})
	It("test component version methods", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(data), &cdref))

		r := &RegistryAccess{
			octx:    ocm.DefaultContext(),
			session: ocm.NewSession(datacontext.NewSession()),
		}
		cv := Must(r.GetComponentVersion(context.Background(), cdref))

		Expect(cv.GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(cv.GetVersion()).To(Equal("1.0.0"))
		Expect(Must(cv.GetComponentDescriptor()).GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(Must(cv.GetComponentDescriptor()).GetVersion()).To(Equal("1.0.0"))
	})

})
