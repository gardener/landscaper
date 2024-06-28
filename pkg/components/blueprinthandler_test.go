// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"context"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/gardener/landscaper/pkg/components/testutils"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
)

var (
	LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS = "./testdata/localcnudierepos/components-with-blueprints"
	LOCALOCMREPOPATH_WITH_BLUEPRINTS    = "./testdata/localocmrepos/components-with-blueprints"

	blueprintsComponentVersionKey = types.ComponentVersionKey{
		Name:    "example.com/landscaper-component-with-blueprints",
		Version: "1.0.0",
	}
)

var _ = Describe("facade implementation compatibility tests", func() {
	var (
		octx ocm.Context
		ctx  context.Context
	)
	// ocmlog.Context().AddRule(logging.NewConditionRule(logging.TraceLevel))

	ocmfactory := &ocmlib.Factory{}
	blueprintData := struct {
		blueprintYaml []byte
		test          []byte
	}{
		blueprintYaml: Must(vfs.ReadFile(osfs.New(), "testdata/localcnudierepos/components-with-blueprints/blobs/blueprint-dir/blueprint.yaml")),
		test:          Must(vfs.ReadFile(osfs.New(), "testdata/localcnudierepos/components-with-blueprints/blobs/blueprint-dir/data/test")),
	}

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})
	// Blueprint Handler
	// The ocmlib backed implementation can automatically convert a directory to a tar, this functionality is tested
	DescribeTable("resolve blueprint dir", func(factory model.Factory, registryRootPath string) {
		registryAccess, err := testutils.NewLocalRegistryAccess(ctx, registryRootPath)
		Expect(err).ToNot(HaveOccurred())
		compvers := Must(registryAccess.GetComponentVersion(ctx, &blueprintsComponentVersionKey))
		res := Must(compvers.GetResource("blueprint-dir", nil))

		typedContent, err := res.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContent).ToNot(BeNil())

		bp, ok := typedContent.Resource.(*blueprints.Blueprint)
		Expect(ok).To(BeTrue())

		blueprintYaml := Must(vfs.ReadFile(bp.Fs, "blueprint.yaml"))
		test := Must(vfs.ReadFile(bp.Fs, "data/test"))
		Expect(blueprintData.blueprintYaml).To(Equal(blueprintYaml))
		Expect(blueprintData.test).To(Equal(test))
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)

	// Both implementations should be able to resolve blueprints in several representations
	DescribeTable("resolve blueprint", func(factory model.Factory, registryRootPath string) {
		registryAccess, err := testutils.NewLocalRegistryAccess(ctx, registryRootPath)
		Expect(err).ToNot(HaveOccurred())
		compvers := Must(registryAccess.GetComponentVersion(ctx, &blueprintsComponentVersionKey))

		bptar := Must(compvers.GetResource("blueprint-tar", nil))
		typedContentTar, err := bptar.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContentTar).ToNot(BeNil())

		bpTar, ok := typedContentTar.Resource.(*blueprints.Blueprint)
		Expect(ok).To(BeTrue())

		blueprintTarYaml := Must(vfs.ReadFile(bpTar.Fs, "blueprint.yaml"))
		tarTest := Must(vfs.ReadFile(bpTar.Fs, "data/test"))
		Expect(blueprintData.blueprintYaml).To(Equal(blueprintTarYaml))
		Expect(blueprintData.test).To(Equal(tarTest))

		bpgzip := Must(compvers.GetResource("blueprint-tar-gzip", nil))
		typedContentGzip, err := bpgzip.GetTypedContent(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(typedContentGzip).ToNot(BeNil())

		bpGzip, ok := typedContentGzip.Resource.(*blueprints.Blueprint)
		Expect(ok).To(BeTrue())

		blueprintGzipYaml := Must(vfs.ReadFile(bpGzip.Fs, "blueprint.yaml"))
		gzipTest := Must(vfs.ReadFile(bpGzip.Fs, "data/test"))
		Expect(blueprintData.blueprintYaml).To(Equal(blueprintGzipYaml))
		Expect(blueprintData.test).To(Equal(gzipTest))
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)

	DescribeTable("error with corrupted blueprint", func(factory model.Factory, registryRootPath string) {
		registryAccess, err := testutils.NewLocalRegistryAccess(ctx, registryRootPath)
		Expect(err).ToNot(HaveOccurred())
		compvers := Must(registryAccess.GetComponentVersion(ctx, &blueprintsComponentVersionKey))
		res := Must(compvers.GetResource("corrupted-blueprint", nil))

		typedContent, err := res.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContent).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)

	// Here, the error is not that it is not a valid tar
	DescribeTable("error with corrupted blueprint tar", func(factory model.Factory, registryRootPath string) {
		registryAccess, err := testutils.NewLocalRegistryAccess(ctx, registryRootPath)
		Expect(err).ToNot(HaveOccurred())
		compvers := Must(registryAccess.GetComponentVersion(ctx, &blueprintsComponentVersionKey))
		res := Must(compvers.GetResource("corrupted-blueprint-tar", nil))

		typedContent, err := res.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContent).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)
})
