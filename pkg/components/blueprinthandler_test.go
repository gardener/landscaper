// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"context"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/mandelsoft/vfs/pkg/osfs"

	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/runtime"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

var (
	LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS = "./testdata/localcnudierepos/components-with-blueprints"
	LOCALOCMREPOPATH_WITH_BLUEPRINTS    = "./testdata/localocmrepos/components-with-blueprints"

	withBlueprintsComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component-with-blueprints",
  "version": "1.0.0"
}
`
)

var _ = Describe("facade implementation compatibility tests", func() {
	ctx := context.Background()
	ocmfactory := &ocmlib.Factory{}
	cnudiefactory := &cnudie.Factory{}
	blueprintData := struct {
		blueprintYaml []byte
		test          []byte
	}{
		blueprintYaml: Must(vfs.ReadFile(osfs.New(), "testdata/localcnudierepos/components-with-blueprints/blobs/blueprint-dir/blueprint.yaml")),
		test:          Must(vfs.ReadFile(osfs.New(), "testdata/localcnudierepos/components-with-blueprints/blobs/blueprint-dir/data/test")),
	}
	// Blueprint Handler
	// The ocmlib backed implementation can automatically convert a directory to a tar, this functionality is tested
	DescribeTable("resolve blueprint dir", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withBlueprintsComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))
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
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withBlueprintsComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))

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
		Entry("with cnudie and v2 descriptors", model.Factory(cnudiefactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)

	DescribeTable("error with corrupted blueprint", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withBlueprintsComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))
		res := Must(compvers.GetResource("corrupted-blueprint", nil))

		typedContent, err := res.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContent).To(BeNil())
	},
		Entry("with cnudie and v2 descriptors", model.Factory(cnudiefactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)

	// Here, the error is not that it is not a valid tar
	DescribeTable("error with corrupted blueprint tar", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withBlueprintsComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))
		res := Must(compvers.GetResource("corrupted-blueprint-tar", nil))

		typedContent, err := res.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContent).To(BeNil())
	},
		Entry("with cnudie and v2 descriptors", model.Factory(cnudiefactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_BLUEPRINTS),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_BLUEPRINTS),
	)
})
