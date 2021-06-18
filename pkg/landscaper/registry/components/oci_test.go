// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/pkg/utils/tar"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	mock_oci "github.com/gardener/component-cli/ociclient/mock"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComponentsRegistry Test Suite")
}

var _ = Describe("Registry", func() {

	var (
		ctrl      *gomock.Controller
		ociClient *mock_oci.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ociClient = mock_oci.NewMockClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should fetch and return a component descriptor when a valid tar is returned", func() {
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()

		ref := cdv2.ObjectMeta{
			Name:    "example.com/my-comp",
			Version: "0.0.1",
		}
		cdConfigLayerDesc := ocispecv1.Descriptor{
			MediaType: cdoci.ComponentDescriptorConfigMimeType,
			Digest:    "0.1.2",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: cdoci.ComponentDescriptorTarMimeType,
			Digest:    "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Config: cdConfigLayerDesc,
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1").Return(manifest, nil)
		ociClient.EXPECT().Fetch(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1", cdConfigLayerDesc, gomock.Any()).Return(nil).Do(func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) {
			data, err := ioutil.ReadFile("./testdata/comp1/config.json")
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(writer, bytes.NewBuffer(data))
			Expect(err).ToNot(HaveOccurred())
		})
		ociClient.EXPECT().Fetch(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1", cdLayerDesc, gomock.Any()).Return(nil).Do(func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) {
			var buf bytes.Buffer
			Expect(tar.BuildTar(osfs.New(), "./testdata/comp1", &buf)).To(Succeed())
			_, err = io.Copy(writer, &buf)
			Expect(err).ToNot(HaveOccurred())
		})

		_, err = cdClient.Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), ref.Name, ref.Version)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch and return a component descriptor when it is defined as json", func() {
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "example.com/my-comp",
			Version: "0.0.1",
		}
		cdConfigLayerDesc := ocispecv1.Descriptor{
			MediaType: cdoci.ComponentDescriptorConfigMimeType,
			Digest:    "0.1.2",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: cdoci.ComponentDescriptorJSONMimeType,
			Digest:    "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Config: cdConfigLayerDesc,
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1").Return(manifest, nil)
		ociClient.EXPECT().Fetch(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1", cdConfigLayerDesc, gomock.Any()).Return(nil).Do(func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) {
			data, err := ioutil.ReadFile("./testdata/comp1/config.json")
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(writer, bytes.NewBuffer(data))
			Expect(err).ToNot(HaveOccurred())
		})
		ociClient.EXPECT().Fetch(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1", cdLayerDesc, gomock.Any()).Return(nil).Do(func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) {
			data, err := ioutil.ReadFile("./testdata/comp1/component-descriptor.yaml")
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(writer, bytes.NewBuffer(data))
			Expect(err).ToNot(HaveOccurred())
		})

		_, err = cdClient.Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), ref.Name, ref.Version)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should throw an error if the manifest has more layers", func() {
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "example.com/my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: cdoci.ComponentDescriptorTarMimeType,
			Digest:    "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{
				cdLayerDesc,
				{
					Digest: "1.2.3",
				},
			},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1").Return(manifest, nil)

		_, err = cdClient.Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), ref.Name, ref.Version)
		Expect(err).To(HaveOccurred())
	})

	It("should throw an error if the manifest has a unknown type", func() {
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "example.com/my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: "unknown-type",
			Digest:    "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/component-descriptors/example.com/my-comp:0.0.1").Return(manifest, nil)

		_, err = cdClient.Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), ref.Name, ref.Version)
		Expect(err).To(HaveOccurred())
	})

	It("should handle an inline component descriptor correctly", func() {

		ref := cdv2.ObjectMeta{
			Name:    "example.com/test",
			Version: "0.1.2",
		}

		repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/component-descriptors", ""))
		Expect(err).ToNot(HaveOccurred())

		cd := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: cdv2.SchemaVersion,
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta:          ref,
				Provider:            "internal",
				RepositoryContexts:  []*cdv2.UnstructuredTypedObject{&repoCtx},
				ComponentReferences: []cdv2.ComponentReference{},
				Sources:             []cdv2.Source{},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Type:    "blueprint",
							Name:    "test",
							Version: "0.1.2",
						},
						Relation: cdv2.LocalRelation,
						Access:   cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"imageReference": "example.com/image-reference:0.1.2"}),
					},
				},
			},
		}

		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient, &cd)
		Expect(err).ToNot(HaveOccurred())

		ctx := context.Background()
		defer ctx.Done()

		returnedCD, err := cdClient.Resolve(ctx, &repoCtx, ref.Name, ref.Version)
		Expect(err).ToNot(HaveOccurred())
		Expect(*returnedCD).To(Equal(cd))
	})

	It("should parse nested inline component descriptors properly", func() {

		ref := cdv2.ObjectMeta{
			Name:    "example.com/label-cd-test",
			Version: "0.2.3",
		}

		repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/nested-component-descriptor", ""))
		Expect(err).ToNot(HaveOccurred())

		labelCd := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: cdv2.SchemaVersion,
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta:          ref,
				Provider:            "internal",
				RepositoryContexts:  []*cdv2.UnstructuredTypedObject{&repoCtx},
				ComponentReferences: []cdv2.ComponentReference{},
				Sources:             []cdv2.Source{},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Type:    "blueprint",
							Name:    "label-cd-test",
							Version: "0.2.3",
						},
						Relation: cdv2.LocalRelation,
						Access:   cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"type": cdv2.OCIRegistryType, "imageReference": "example.com/image-reference:0.2.3"}),
					},
				},
			},
		}

		labelCdJson, err := codec.Encode(&labelCd)
		Expect(err).ToNot(HaveOccurred())

		nestedRepoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/component-descriptors", ""))
		Expect(err).ToNot(HaveOccurred())
		cd := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: cdv2.SchemaVersion,
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "example.com/root-cd",
					Version: "0.1.2",
				},
				Provider:           "internal",
				RepositoryContexts: []*cdv2.UnstructuredTypedObject{&nestedRepoCtx},
				Sources:            []cdv2.Source{},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Type:    "blueprint",
							Name:    "test",
							Version: "0.1.2",
						},
						Relation: cdv2.LocalRelation,
						Access:   cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"imageReference": "example.com/image-reference:0.1.2"}),
					},
				},
				ComponentReferences: []cdv2.ComponentReference{
					{
						Name:          "label-cd-test",
						ComponentName: "example.com/label-cd-test",
						Version:       "0.2.3",
						Labels: []cdv2.Label{
							{
								Name:  "landscaper.gardener.cloud/component-descriptor",
								Value: labelCdJson,
							},
							{
								Name:  "foo",
								Value: []byte("{\"bar\": \"foo.bar\"}"),
							},
						},
					},
				},
			},
		}
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient, &cd)
		Expect(err).ToNot(HaveOccurred())

		ctx := context.Background()
		defer ctx.Done()

		returnedCD, err := cdClient.Resolve(ctx, &repoCtx, ref.Name, ref.Version)
		Expect(err).ToNot(HaveOccurred())
		err = codec.Decode(labelCdJson, &labelCd)
		Expect(err).ToNot(HaveOccurred())
		Expect(labelCd).To(Equal(*returnedCD))
	})

	It("should parse two levels of nested inline component descriptors", func() {
		//declare lvl2
		refLvl2 := cdv2.ObjectMeta{
			Name:    "example.com/label-cd-lvl-2",
			Version: "0.2.3",
		}

		repoCtxLvl2, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/nested-component-descriptor", ""))
		Expect(err).ToNot(HaveOccurred())

		labelCdLvl2 := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: cdv2.SchemaVersion,
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta:          refLvl2,
				Provider:            "internal",
				RepositoryContexts:  []*cdv2.UnstructuredTypedObject{&repoCtxLvl2},
				ComponentReferences: []cdv2.ComponentReference{},
				Sources:             []cdv2.Source{},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Type:    "blueprint",
							Name:    "label-cd-lvl2",
							Version: "0.2.3",
						},
						Relation: cdv2.LocalRelation,
						Access:   cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"type": cdv2.OCIRegistryType, "imageReference": "example.com/image-reference:0.2.3"}),
					},
				},
			},
		}

		labelCdLvl2Json, err := codec.Encode(&labelCdLvl2)
		Expect(err).ToNot(HaveOccurred())

		//declare lvl1
		refLvl1 := cdv2.ObjectMeta{
			Name:    "example.com/label-cd-lvl1",
			Version: "0.1.2",
		}

		repoCtxLvl1, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/nested-component-descriptor", ""))
		Expect(err).ToNot(HaveOccurred())

		labelCdLvl1 := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: cdv2.SchemaVersion,
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta:         refLvl1,
				Provider:           "internal",
				RepositoryContexts: []*cdv2.UnstructuredTypedObject{&repoCtxLvl1},
				Sources:            []cdv2.Source{},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Type:    "blueprint",
							Name:    "label-cd-lvl1",
							Version: "0.1.2",
						},
						Relation: cdv2.LocalRelation,
						Access:   cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"type": cdv2.OCIRegistryType, "imageReference": "example.com/image-reference:0.1.2"}),
					},
				},
				ComponentReferences: []cdv2.ComponentReference{
					{
						Name:          "label-cd-lvl2",
						ComponentName: "example.com/label-cd-lvl2",
						Version:       "0.2.3",
						Labels: []cdv2.Label{
							{
								Name:  "landscaper.gardener.cloud/component-descriptor",
								Value: labelCdLvl2Json,
							},
							{
								Name:  "foo",
								Value: []byte("{\"bar\": \"foo.bar\"}"),
							},
						},
					},
				},
			},
		}

		labelCdLvl1Json, err := codec.Encode(&labelCdLvl1)
		Expect(err).ToNot(HaveOccurred())

		//declare inlineCD

		refRootCd := cdv2.ObjectMeta{
			Name:    "example.com/root-cd",
			Version: "0.1.2",
		}

		repoCtxRootCd, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/component-descriptors", ""))
		Expect(err).ToNot(HaveOccurred())

		cd := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: cdv2.SchemaVersion,
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta:         refRootCd,
				Provider:           "internal",
				RepositoryContexts: []*cdv2.UnstructuredTypedObject{&repoCtxRootCd},
				Sources:            []cdv2.Source{},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Type:    "blueprint",
							Name:    "root",
							Version: "0.1.2",
						},
						Relation: cdv2.LocalRelation,
						Access:   cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"imageReference": "example.com/image-reference:0.1.2"}),
					},
				},
				ComponentReferences: []cdv2.ComponentReference{
					{
						Name:          "label-cd-lvl1",
						ComponentName: "example.com/label-cd-lvl1",
						Version:       "0.1.2",
						Labels: []cdv2.Label{
							{
								Name:  "landscaper.gardener.cloud/component-descriptor",
								Value: labelCdLvl1Json,
							},
							{
								Name:  "foo",
								Value: []byte("{\"bar\": \"foo.bar\"}"),
							},
						},
					},
				},
			},
		}

		// parse component descriptor
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logr.Discard(), ociClient, &cd)
		Expect(err).ToNot(HaveOccurred())

		ctx := context.Background()
		defer ctx.Done()

		//resolve levels
		returnedCDLvl2, err := cdClient.Resolve(ctx, &repoCtxLvl2, refLvl2.Name, refLvl2.Version)
		Expect(err).ToNot(HaveOccurred())
		err = codec.Decode(labelCdLvl2Json, &labelCdLvl2)
		Expect(err).ToNot(HaveOccurred())
		Expect(labelCdLvl2).To(Equal(*returnedCDLvl2))

		returnedCDLvl1, err := cdClient.Resolve(ctx, &repoCtxLvl1, refLvl1.Name, refLvl1.Version)
		Expect(err).ToNot(HaveOccurred())
		err = codec.Decode(labelCdLvl1Json, &labelCdLvl1)
		Expect(err).ToNot(HaveOccurred())
		Expect(labelCdLvl1).To(Equal(*returnedCDLvl1))

		returnedCDRoot, err := cdClient.Resolve(ctx, &repoCtxRootCd, refRootCd.Name, refRootCd.Version)
		Expect(err).ToNot(HaveOccurred())
		Expect(cd).To(Equal(*returnedCDRoot))
	})

})
