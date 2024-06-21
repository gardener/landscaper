// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"context"
	"path/filepath"
	"reflect"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
)

var (
	BLUEPRINT_RESOURCE_NAME   = "blueprint"
	GENERIC_RESOURCE_NAME     = "genericresource"
	REFERENCED_COMPONENT_NAME = "referenced-landscaper-component"

	LOCALCNUDIEREPOPATH_VALID = "./testdata/localcnudierepos/valid-components"
	LOCALOCMREPOPATH_VALID    = "./testdata/localocmrepos/valid-components"

	LOCALCNUDIEREPOPATH_WITHOUT_REPOCTX = "./testdata/localcnudierepos/components-without-repoctx"
	LOCALOCMREPOPATH_WITHOUT_REPOCTX    = "./testdata/localocmrepos/components-without-repoctx"

	LOCALCNUDIEREPOPATH_WITH_INVALID_ACCESS_TYPE = "./testdata/localcnudierepos/components-with-invalid-access-type"
	LOCALOCMREPOPATH_WITH_INVALID_ACCESS_TYPE    = "./testdata/localocmrepos/components-with-invalid-access-type"

	LOCALCNUDIEREPOPATH_WITH_INVALID_REFERENCE = "./testdata/localcnudierepos/components-with-invalid-reference"
	LOCALOCMREPOPATH_WITH_INVALID_REFERENCE    = "./testdata/localocmrepos/components-with-invalid-reference"

	LOCALCNUDIEREPOPATH_WITH_INVALID_COMPONENT = "./testdata/localcnudierepos/invalid-components"
	LOCALOCMREPOPATH_WITH_INVALID_COMPONENT    = "./testdata/localocmrepos/invalid-components"

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

	referencedComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/referenced-landscaper-component",
  "version": "1.0.0"
}
`

	withoutRepoctxComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component-without-repository-context",
  "version": "1.0.0"
}
`

	withInvalidAccessTypeComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component-with-invalid-access-type",
  "version": "1.0.0"
}
`

	withInvalidReferenceComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component-with-invalid-reference",
  "version": "1.0.0"
}
`

	invalidComponentComponentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/invalid-landscaper-component",
  "version": "1.0.0"
}
`

	repositoryContext = `
{
    "type": "local",
	"filePath": "./"
}
`
	inlineRepoCtxCnudie = `
{
	"type": "ociRegistry",
	"baseUrl": "/"
}
`

	inlineRepoCtxOCM = `
{
	"type": "ociRegistry",
	"baseUrl": "/"
}
`
)

// These test shall ensure that all facade implementations and component descriptor versions have the same behavior.
// If this can be guaranteed, the rest of the landscaper as well as the other tests should not have to care about the
// actual facade implementation.
var _ = Describe("facade implementation compatibility tests", func() {
	var (
		ctx  context.Context
		octx ocm.Context
	)
	ocmfactory := &ocmlib.Factory{}

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	// Test for the expected "straight forward" cases
	It("compatibility of facade implementations and component descriptor versions", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), cdref))

		oRaForCnudie := Must(ocmfactory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH_VALID}, nil, nil))
		oRaForOcm := Must(ocmfactory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALOCMREPOPATH_VALID}, nil, nil))

		// the 3 registry accesses should all behave the same and the interface methods should return the same data
		oRaForCnudieCv := Must(oRaForCnudie.GetComponentVersion(ctx, cdref))
		oRaForOcmCv := Must(oRaForOcm.GetComponentVersion(ctx, cdref))

		Expect(oRaForCnudieCv.GetName()).To(Equal(oRaForOcmCv.GetName()))

		Expect(oRaForCnudieCv.GetVersion()).To(Equal(oRaForOcmCv.GetVersion()))

		Expect(oRaForCnudieCv.GetComponentDescriptor()).To(Equal(oRaForOcmCv.GetComponentDescriptor()))

		Expect(oRaForCnudieCv.GetComponentReferences()).To(Equal(oRaForOcmCv.GetComponentReferences()))

		Expect(oRaForCnudieCv.GetComponentReference(REFERENCED_COMPONENT_NAME)).To(Equal(oRaForOcmCv.GetComponentReference(REFERENCED_COMPONENT_NAME)))

		repoCtx := &cdv2.UnstructuredTypedObject{}
		Expect(repoCtx.UnmarshalJSON([]byte(repositoryContext))).To(Succeed())

		oRaForCnudieRefCv := Must(oRaForCnudieCv.GetReferencedComponentVersion(ctx, oRaForCnudieCv.GetComponentReference(REFERENCED_COMPONENT_NAME), repoCtx, nil))
		oRaForOcmRefCv := Must(oRaForOcmCv.GetReferencedComponentVersion(ctx, oRaForOcmCv.GetComponentReference(REFERENCED_COMPONENT_NAME), repoCtx, nil))
		Expect(reflect.DeepEqual(oRaForCnudieRefCv.GetComponentDescriptor(), oRaForOcmRefCv.GetComponentDescriptor()))

		oRaForCnudieRs := Must(oRaForCnudieCv.GetResource(BLUEPRINT_RESOURCE_NAME, nil))
		oRaForOcmRs := Must(oRaForOcmCv.GetResource(BLUEPRINT_RESOURCE_NAME, nil))

		Expect(oRaForCnudieRs.GetName()).To(Equal(oRaForOcmRs.GetName()))

		Expect(oRaForCnudieRs.GetType()).To(Equal(oRaForOcmRs.GetType()))

		Expect(oRaForCnudieRs.GetVersion()).To(Equal(oRaForOcmRs.GetVersion()))

		Expect(oRaForCnudieRs.GetAccessType()).To(Equal(oRaForOcmRs.GetAccessType()))

		res1 := Must(oRaForCnudieRs.GetResource())
		res2 := Must(oRaForOcmRs.GetResource())

		blueprint1 := Must(oRaForCnudieRs.GetTypedContent(ctx)).Resource.(*blueprints.Blueprint)
		blueprint2 := Must(oRaForOcmRs.GetTypedContent(ctx)).Resource.(*blueprints.Blueprint)
		Expect(Must(vfs.ReadFile(blueprint1.Fs, filepath.Join("/blueprint.yaml")))).To(Equal(Must(vfs.ReadFile(blueprint2.Fs, filepath.Join("/blueprint.yaml")))))

		// ignore raw value as the order of the values might vary
		res1.Access.Raw = []byte{}
		res2.Access.Raw = []byte{}
		Expect(reflect.DeepEqual(res1, res2))
	})

	// Resolve component referenced by an inline component descriptor
	// Resolving of references cannot really be unit tested as the component descriptors v2 mandate that the
	// repositoryContext is of type ociRegistry, but this is also the context that is evaluated to resolve the reference
	DescribeTable("resolve reference based on the inline descriptors repository context", func(factory model.Factory, repoCtx string) {
		rctx := &cdv2.UnstructuredTypedObject{}
		Expect(rctx.UnmarshalJSON([]byte(repoCtx))).To(Succeed())

		inlineDescriptor := &types.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: "v2",
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "example.com/inline-component-descriptor",
					Version: "1.0.0",
				},
				RepositoryContexts:  []*cdv2.UnstructuredTypedObject{rctx},
				Provider:            "internal",
				Sources:             []cdv2.Source{},
				ComponentReferences: []cdv2.ComponentReference{},
				Resources:           []cdv2.Resource{},
			},
			Signatures: nil,
		}

		cdref := &v1alpha1.ComponentDescriptorReference{
			RepositoryContext: rctx,
			ComponentName:     "example.com/inline-component-descriptor",
			Version:           "1.0.0",
		}

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: "./"}, nil, inlineDescriptor))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))
		Expect(compvers.GetComponentDescriptor()).To(YAMLEqual(inlineDescriptor))
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), inlineRepoCtxCnudie),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), inlineRepoCtxOCM),
	)

	DescribeTable("error when trying to access unknown resource", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(referencedComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))
		res, err := compvers.GetResource("non-existent-resource", nil)
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_VALID),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_VALID),
	)

	// Unknown resource type is an important test here (in the facade implementation) since the facade actually
	// implements against the resource type.
	DescribeTable("error when accessing resources with unknown resource type", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(referencedComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers := Must(registryAccess.GetComponentVersion(ctx, cdref))
		res := Must(compvers.GetResource(GENERIC_RESOURCE_NAME, nil))

		typedContent, err := res.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(typedContent).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_VALID),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_VALID),
	)

	// This is due to compatibility
	// Theoretically, a component descriptor (and consequently a component version) does not have to have a repository
	// context (as per ocm spec)
	DescribeTable("error when component descriptor has no repository context", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withoutRepoctxComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers, err := registryAccess.GetComponentVersion(ctx, cdref)
		Expect(err).To(HaveOccurred())
		Expect(compvers).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITHOUT_REPOCTX),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITHOUT_REPOCTX),
	)

	DescribeTable("error when component descriptor has invalid access type", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withInvalidAccessTypeComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers, err := registryAccess.GetComponentVersion(ctx, cdref)
		Expect(err).ToNot(HaveOccurred())
		Expect(compvers).ToNot(BeNil())

		res, err := compvers.GetResource(GENERIC_RESOURCE_NAME, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())

		content, err := res.GetTypedContent(ctx)
		Expect(err).To(HaveOccurred())
		Expect(content).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_INVALID_ACCESS_TYPE),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_INVALID_ACCESS_TYPE),
	)

	DescribeTable("error when component descriptor has invalid reference", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(withInvalidReferenceComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers, err := registryAccess.GetComponentVersion(ctx, cdref)
		Expect(err).ToNot(HaveOccurred())
		Expect(compvers).ToNot(BeNil())

		ref := compvers.GetComponentReference("invalid-component-reference")
		Expect(ref).ToNot(BeNil())

		repoCtx := &cdv2.UnstructuredTypedObject{}
		Expect(repoCtx.UnmarshalJSON([]byte(repositoryContext))).To(Succeed())

		referencedComponent, err := compvers.GetReferencedComponentVersion(ctx, ref, repoCtx, nil)
		Expect(err).To(HaveOccurred())
		Expect(referencedComponent).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_INVALID_REFERENCE),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_INVALID_REFERENCE),
	)

	// Component Descriptors v2 and v3 are validated against different json schemas, so inevitably, here is a slight
	// difference between the facade implementations.
	// This check shall check the general handling of an invalid component descriptor.
	DescribeTable("error when component descriptor is invalid (does not adhere to its json schema)", func(factory model.Factory, registryRootPath string) {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(invalidComponentComponentReference), cdref))

		registryAccess := Must(factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil))
		compvers, err := registryAccess.GetComponentVersion(ctx, cdref)
		Expect(err).To(HaveOccurred())
		Expect(compvers).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_WITH_INVALID_COMPONENT),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_WITH_INVALID_COMPONENT),
	)

	// Check nil argument handling of facade methods
	DescribeTable("prevent null pointer exceptions", func(factory model.Factory, registryRootPath string) {
		// Test registry access
		registryAccess, err := factory.NewRegistryAccess(ctx, nil, nil, nil, nil, nil, nil, nil)
		Expect(registryAccess).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())

		compvers, err := registryAccess.GetComponentVersion(ctx, nil)
		Expect(compvers).To(BeNil())
		Expect(err).To(HaveOccurred())

		// Organize a valid component version
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), cdref))
		registryAccess, err = factory.NewRegistryAccess(ctx, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: registryRootPath}, nil, nil)
		Expect(registryAccess).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())

		compvers = Must(registryAccess.GetComponentVersion(ctx, cdref))

		// Test component version
		res, err := compvers.GetResource("", nil)
		Expect(res).To(BeNil())
		Expect(err).To(HaveOccurred())

		referencedComponent, err := compvers.GetReferencedComponentVersion(ctx, nil, nil, nil)
		Expect(err).To(HaveOccurred())
		Expect(referencedComponent).To(BeNil())
	},
		Entry("with ocm and v2 descriptors", model.Factory(ocmfactory), LOCALCNUDIEREPOPATH_VALID),
		Entry("with ocm and v3 descriptors", model.Factory(ocmfactory), LOCALOCMREPOPATH_VALID),
	)
})
