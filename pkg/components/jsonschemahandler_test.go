// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/runtime"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/pkg/components/model"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
)

var (
	LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS = "./testdata/localcnudierepos/components-with-jsonschemas"
	LOCALOCMREPOPATH_WITH_JSONSCHEMAS    = "./testdata/localocmrepos/components-with-jsonschemas"

	withJSONSchemasComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component-with-jsonschemas",
  "version": "1.0.0"
}
`
)

var _ = Describe("facade implementation compatibility tests", func() {
	var (
		ctx  context.Context
		octx ocm.Context
	)
	ocmfactory := &ocmlib.Factory{}
	cnudiefactory := &cnudie.Factory{}
	jsonschemaData := Must(vfs.ReadFile(osfs.New(), "testdata/localcnudierepos/components-with-jsonschemas/blobs/jsonschema.json"))

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})

	// JsonSchema Handler
	DescribeTable("resolve jsonschema", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withJSONSchemasComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))

		jsonschema := Must(compvers.GetResource("jsonschema", nil))
		typedContent, err := jsonschema.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContent).ToNot(BeNil())

		schema, ok := typedContent.Resource.([]byte)
		Expect(ok).To(BeTrue())
		Expect(jsonschemaData).To(Equal(schema))

		jsonschemagzip := Must(compvers.GetResource("jsonschema-compressed", nil))
		typedContentGzip, err := jsonschemagzip.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContentGzip).ToNot(BeNil())

		schemaGzip, ok := typedContentGzip.Resource.([]byte)
		Expect(ok).To(BeTrue())
		Expect(jsonschemaData).To(Equal(schemaGzip))
	},
		Entry("with cnudie and v2 descriptors", model.Factory(cnudiefactory), LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS),
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_JSONSCHEMAS),
	)

	DescribeTable("resolve jsonschema with unknown mediatype with ocmlib facade implementation", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withJSONSchemasComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))

		jsonschemaUnknown := Must(compvers.GetResource("jsonschema-unknown-mediatype", nil))
		typedContentUnknown, err := jsonschemaUnknown.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContentUnknown).ToNot(BeNil())

		schemaUnknown, ok := typedContentUnknown.Resource.([]byte)
		Expect(ok).To(BeTrue())
		Expect(jsonschemaData).To(Equal(schemaUnknown))

		jsonschemagzip := Must(compvers.GetResource("jsonschema-compressed-unknown-mediatype", nil))
		typedContentGzip, err := jsonschemagzip.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContentGzip).ToNot(BeNil())

		schemaGzip, ok := typedContentGzip.Resource.([]byte)
		Expect(ok).To(BeTrue())
		Expect(jsonschemaData).To(Equal(schemaGzip))
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_JSONSCHEMAS),
	)

	DescribeTable("error when resolving jsonschema with unknown mediatype with component-cli facade implementation", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withJSONSchemasComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))

		jsonschemaUnknown := Must(compvers.GetResource("jsonschema-unknown-mediatype", nil))
		typedContentUnknown, err := jsonschemaUnknown.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContentUnknown).To(BeNil())

		jsonschemagzip := Must(compvers.GetResource("jsonschema-compressed-unknown-mediatype", nil))
		typedContentGzip, err := jsonschemagzip.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContentGzip).To(BeNil())
	},
		Entry("with cnudie and v2 descriptors", model.Factory(cnudiefactory), LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS),
	)
})
