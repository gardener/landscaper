// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"context"

	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/testutils"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/pkg/components/model"

	"github.com/gardener/landscaper/pkg/components/ocmlib"
)

var (
	LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS = "./testdata/localcnudierepos/components-with-jsonschemas"
	LOCALOCMREPOPATH_WITH_JSONSCHEMAS    = "./testdata/localocmrepos/components-with-jsonschemas"

	jsonSchemaComponentVersionKey = types.ComponentVersionKey{
		Name:    "example.com/landscaper-component-with-jsonschemas",
		Version: "1.0.0",
	}
)

var _ = Describe("facade implementation compatibility tests", func() {
	var (
		ctx  context.Context
		octx ocm.Context
	)
	ocmfactory := &ocmlib.Factory{}
	jsonschemaData := Must(vfs.ReadFile(osfs.New(), "testdata/localcnudierepos/components-with-jsonschemas/blobs/jsonschema.json"))

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})

	// JsonSchema Handler
	DescribeTable("resolve jsonschema", func(factory model.Factory, registryRootPath string) {
		registryAccess, err := testutils.NewLocalRegistryAccess(ctx, registryRootPath)
		Expect(err).ToNot(HaveOccurred())

		compvers := Must(registryAccess.GetComponentVersion(ctx, &jsonSchemaComponentVersionKey))

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
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_JSONSCHEMAS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_JSONSCHEMAS),
	)

	DescribeTable("resolve jsonschema with unknown mediatype with ocmlib facade implementation", func(factory model.Factory, registryRootPath string) {
		registryAccess, err := testutils.NewLocalRegistryAccess(ctx, registryRootPath)
		Expect(err).ToNot(HaveOccurred())

		compvers := Must(registryAccess.GetComponentVersion(ctx, &jsonSchemaComponentVersionKey))

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
})
