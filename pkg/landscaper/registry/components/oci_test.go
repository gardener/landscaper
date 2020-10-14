// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry_test

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	logtesting "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils"
	mock_oci "github.com/gardener/landscaper/pkg/utils/oci/mock"
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
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logtesting.NullLogger{}, ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: componentsregistry.ComponentDescriptorMediaType,
			Digest:    "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/my-comp:0.0.1").Return(manifest, nil)
		ociClient.EXPECT().Fetch(ctx, "example.com/my-comp:0.0.1", cdLayerDesc, gomock.Any()).Return(nil).Do(func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) {
			var buf bytes.Buffer
			absPath, err := filepath.Abs("./testdata/comp1")
			Expect(err).ToNot(HaveOccurred())
			Expect(utils.BuildTar(osfs.New(), absPath, &buf)).To(Succeed())
			_, err = io.Copy(writer, &buf)
			Expect(err).ToNot(HaveOccurred())
		})

		_, err = cdClient.Resolve(ctx, cdv2.RepositoryContext{Type: cdv2.OCIRegistryType, BaseURL: "example.com"}, ref)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should throw an error if the manifest has more layers", func() {
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logtesting.NullLogger{}, ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: componentsregistry.ComponentDescriptorMediaType,
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

		ociClient.EXPECT().GetManifest(ctx, "example.com/my-comp:0.0.1").Return(manifest, nil)

		_, err = cdClient.Resolve(ctx, cdv2.RepositoryContext{Type: cdv2.OCIRegistryType, BaseURL: "example.com"}, ref)
		Expect(err).To(HaveOccurred())
	})

	It("should throw an error if the manifest has a unknown type", func() {
		cdClient, err := componentsregistry.NewOCIRegistryWithOCIClient(logtesting.NullLogger{}, ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: "unknown-type",
			Digest:    "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/my-comp:0.0.1").Return(manifest, nil)

		_, err = cdClient.Resolve(ctx, cdv2.RepositoryContext{Type: cdv2.OCIRegistryType, BaseURL: "example.com"}, ref)
		Expect(err).To(HaveOccurred())
	})

})
