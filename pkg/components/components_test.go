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
	"github.com/open-component-model/ocm/pkg/runtime"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

var (
	RESOURCE_NAME             = "blueprint"
	REFERENCED_COMPONENT_NAME = "referenced-landscaper-component"

	LOCALCNUDIEREPOPATH = "./testdata/localcnudierepo"
	LOCALOCMREPOPATH    = "./testdata/localocmrepo"

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

	repositoryContext = `
{
    "type": "local",
	"filePath": "./"
}
`
)

var _ = Describe("facade implementation compatibility tests", func() {
	ctx := context.Background()
	ocmfactory := ocmlib.Factory{}
	cnudiefactory := cnudie.Factory{}

	It("compatibility of facade implementations and component descriptor versions", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), cdref))

		// localFSAccess := Must(componentresolvers.NewLocalFilesystemBlobAccess("bp.tar", ""))
		oRaForCnudie := Must(ocmfactory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH}, nil, nil))
		oRaForOcm := Must(ocmfactory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALOCMREPOPATH}, nil, nil))
		cnudieRa := Must(cnudiefactory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH}, nil, nil))

		// the 3 registry accesses should all behave the same and the interface methods should return the same data
		oRaForCnudieCv := Must(oRaForCnudie.GetComponentVersion(ctx, cdref))
		oRaForOcmCv := Must(oRaForOcm.GetComponentVersion(ctx, cdref))
		cnudieRaCv := Must(cnudieRa.GetComponentVersion(ctx, cdref))

		Expect(oRaForCnudieCv.GetName()).To(Equal(oRaForOcmCv.GetName()))
		Expect(oRaForCnudieCv.GetName()).To(Equal(cnudieRaCv.GetName()))

		Expect(oRaForCnudieCv.GetVersion()).To(Equal(oRaForOcmCv.GetVersion()))
		Expect(oRaForCnudieCv.GetVersion()).To(Equal(cnudieRaCv.GetVersion()))

		Expect(oRaForCnudieCv.GetComponentDescriptor()).To(Equal(oRaForOcmCv.GetComponentDescriptor()))
		Expect(oRaForCnudieCv.GetComponentDescriptor()).To(Equal(cnudieRaCv.GetComponentDescriptor()))

		Expect(oRaForCnudieCv.GetRepositoryContext()).To(Equal(oRaForOcmCv.GetRepositoryContext()))
		Expect(oRaForCnudieCv.GetRepositoryContext()).To(Equal(cnudieRaCv.GetRepositoryContext()))

		Expect(oRaForCnudieCv.GetComponentReferences()).To(Equal(oRaForOcmCv.GetComponentReferences()))
		Expect(oRaForCnudieCv.GetComponentReferences()).To(Equal(cnudieRaCv.GetComponentReferences()))

		Expect(oRaForCnudieCv.GetComponentReference(REFERENCED_COMPONENT_NAME)).To(Equal(oRaForOcmCv.GetComponentReference(REFERENCED_COMPONENT_NAME)))
		Expect(oRaForCnudieCv.GetComponentReference(REFERENCED_COMPONENT_NAME)).To(Equal(cnudieRaCv.GetComponentReference(REFERENCED_COMPONENT_NAME)))

		repoCtx := &cdv2.UnstructuredTypedObject{}
		Expect(repoCtx.UnmarshalJSON([]byte(repositoryContext))).To(Succeed())

		oRaForCnudieRefCv := Must(oRaForCnudieCv.GetReferencedComponentVersion(ctx, oRaForCnudieCv.GetComponentReference(REFERENCED_COMPONENT_NAME), repoCtx, nil))
		oRaForOcmRefCv := Must(oRaForOcmCv.GetReferencedComponentVersion(ctx, oRaForOcmCv.GetComponentReference(REFERENCED_COMPONENT_NAME), repoCtx, nil))
		cnudieRaRefCv := Must(cnudieRaCv.GetReferencedComponentVersion(ctx, cnudieRaCv.GetComponentReference(REFERENCED_COMPONENT_NAME), repoCtx, nil))
		Expect(reflect.DeepEqual(oRaForCnudieRefCv.GetComponentDescriptor(), oRaForOcmRefCv.GetComponentDescriptor()))
		Expect(reflect.DeepEqual(oRaForCnudieRefCv.GetComponentDescriptor(), cnudieRaRefCv.GetComponentDescriptor()))

		oRaForCnudieRs := Must(oRaForCnudieCv.GetResource(RESOURCE_NAME, nil))
		oRaForOcmRs := Must(oRaForOcmCv.GetResource(RESOURCE_NAME, nil))
		cnudieRaRs := Must(cnudieRaCv.GetResource(RESOURCE_NAME, nil))

		Expect(oRaForCnudieRs.GetName()).To(Equal(oRaForOcmRs.GetName()))
		Expect(oRaForCnudieRs.GetName()).To(Equal(cnudieRaRs.GetName()))

		Expect(oRaForCnudieRs.GetType()).To(Equal(oRaForOcmRs.GetType()))
		Expect(oRaForCnudieRs.GetType()).To(Equal(cnudieRaRs.GetType()))

		Expect(oRaForCnudieRs.GetVersion()).To(Equal(oRaForOcmRs.GetVersion()))
		Expect(oRaForCnudieRs.GetVersion()).To(Equal(cnudieRaRs.GetVersion()))

		Expect(oRaForCnudieRs.GetAccessType()).To(Equal(oRaForOcmRs.GetAccessType()))
		Expect(oRaForCnudieRs.GetAccessType()).To(Equal(cnudieRaRs.GetAccessType()))

		res1 := Must(oRaForCnudieRs.GetResource())
		res2 := Must(oRaForOcmRs.GetResource())
		res3 := Must(cnudieRaRs.GetResource())

		blueprint1 := Must(oRaForCnudieRs.GetTypedContent(ctx)).Resource.(*blueprints.Blueprint)
		blueprint2 := Must(oRaForOcmRs.GetTypedContent(ctx)).Resource.(*blueprints.Blueprint)
		blueprint3 := Must(cnudieRaRs.GetTypedContent(ctx)).Resource.(*blueprints.Blueprint)
		Expect(Must(vfs.ReadFile(blueprint1.Fs, filepath.Join("/blueprint.yaml")))).To(Equal(Must(vfs.ReadFile(blueprint2.Fs, filepath.Join("/blueprint.yaml")))))
		Expect(Must(vfs.ReadFile(blueprint1.Fs, filepath.Join("/blueprint.yaml")))).To(Equal(Must(vfs.ReadFile(blueprint3.Fs, filepath.Join("/blueprint.yaml")))))

		// ignore raw value as the order of the values might vary
		res1.Access.Raw = []byte{}
		res2.Access.Raw = []byte{}
		res3.Access.Raw = []byte{}
		Expect(reflect.DeepEqual(res1, res2))
		Expect(reflect.DeepEqual(res1, res3))
	})
})
