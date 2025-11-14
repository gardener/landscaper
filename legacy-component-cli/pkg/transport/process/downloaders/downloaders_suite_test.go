// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/credentials"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/oci"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/test/envtest"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/testutils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Downloaders Test Suite")
}

var (
	testenv               *envtest.Environment
	ociClient             ociclient.Client
	ociCache              cache.Cache
	keyring               *credentials.GeneralOciKeyring
	testComponent         cdv2.ComponentDescriptor
	localOciBlobResIndex  = 0
	localOciBlobData      = []byte("Hello World")
	imageRef              string
	imageResIndex         = 1
	expectedImageManifest oci.Manifest
	imageIndexRef         string
	imageIndexResIndex    = 2
	expectedImageIndex    oci.Index
)

var _ = ginkgo.BeforeSuite(func() {
	testenv = envtest.New(envtest.Options{
		RegistryBinaryPath: filepath.Join("../../../../", envtest.DefaultRegistryBinaryPath),
		Stdout:             ginkgo.GinkgoWriter,
		Stderr:             ginkgo.GinkgoWriter,
	})
	Expect(testenv.Start(context.Background())).To(Succeed())

	keyring = credentials.New()
	Expect(keyring.AddAuthConfig(testenv.Addr, credentials.AuthConfig{
		Username: testenv.BasicAuth.Username,
		Password: testenv.BasicAuth.Password,
	})).To(Succeed())
	ociCache = cache.NewInMemoryCache()
	var err error
	ociClient, err = ociclient.NewClient(logr.Discard(), ociclient.WithKeyring(keyring), ociclient.WithCache(ociCache))
	Expect(err).ToNot(HaveOccurred())

	uploadTestComponent()
}, 60)

var _ = ginkgo.AfterSuite(func() {
	Expect(testenv.Close()).To(Succeed())
})

func uploadTestComponent() {
	ctx := context.TODO()
	fs := memoryfs.New()

	localOciBlobRes := createLocalOciBlobRes(fs)
	imageRes := createImageRes(ctx)
	imageIndexRes := createImageIndexRes(ctx)

	ociRepo := cdv2.NewOCIRegistryRepository(testenv.Addr+"/test/downloaders", "")
	repoCtx, err := cdv2.NewUnstructured(
		ociRepo,
	)
	Expect(err).ToNot(HaveOccurred())

	localCd := cdv2.ComponentDescriptor{
		ComponentSpec: cdv2.ComponentSpec{
			ObjectMeta: cdv2.ObjectMeta{
				Name:    "github.com/component-cli/test-component",
				Version: "0.1.0",
			},
			Provider: "internal",
			RepositoryContexts: []*cdv2.UnstructuredTypedObject{
				&repoCtx,
			},
			Resources: []cdv2.Resource{
				localOciBlobRes,
				imageRes,
				imageIndexRes,
			},
		},
	}

	manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(&localCd, fs)).Build(ctx)
	Expect(err).ToNot(HaveOccurred())

	ociRef, err := cdoci.OCIRef(*ociRepo, localCd.Name, localCd.Version)
	Expect(err).ToNot(HaveOccurred())

	Expect(ociClient.PushManifest(ctx, ociRef, manifest)).To(Succeed())

	cdresolver := cdoci.NewResolver(ociClient)
	actualCd, err := cdresolver.Resolve(ctx, ociRepo, localCd.Name, localCd.Version)
	Expect(err).ToNot(HaveOccurred())

	testComponent = *actualCd
}

func createLocalOciBlobRes(fs vfs.FileSystem) cdv2.Resource {
	Expect(fs.Mkdir(ctf.BlobsDirectoryName, os.ModePerm)).To(Succeed())

	dgst := digest.FromBytes(localOciBlobData)
	blobfile, err := fs.Create(ctf.BlobPath(dgst.String()))
	Expect(err).ToNot(HaveOccurred())

	_, err = blobfile.Write(localOciBlobData)
	Expect(err).ToNot(HaveOccurred())

	Expect(blobfile.Close()).To(Succeed())

	localOciBlobAcc, err := cdv2.NewUnstructured(
		cdv2.NewLocalFilesystemBlobAccess(
			dgst.String(),
			"text/plain",
		),
	)
	Expect(err).ToNot(HaveOccurred())

	localOciBlobRes := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "local-oci-blob",
			Version: "0.1.0",
			Type:    "plain-text",
		},
		Relation: cdv2.LocalRelation,
		Access:   &localOciBlobAcc,
	}

	return localOciBlobRes
}

func createImageRes(ctx context.Context) cdv2.Resource {
	imageRef = testenv.Addr + "/test/downloaders/image:0.1.0"

	configData := []byte("config-data")
	layersData := [][]byte{
		[]byte("layer-data"),
	}

	mdesc, mbytes := testutils.UploadTestImage(ctx, ociClient, imageRef, ocispecv1.MediaTypeImageManifest, configData, layersData)
	testutils.CompareRemoteManifest(ctx, ociClient, imageRef, mdesc, mbytes, configData, layersData)

	manifest := ocispecv1.Manifest{}
	Expect(json.Unmarshal(mbytes, &manifest)).To(Succeed())

	expectedImageManifest = oci.Manifest{
		Descriptor: mdesc,
		Data:       &manifest,
	}

	acc, err := cdv2.NewUnstructured(
		cdv2.NewOCIRegistryAccess(
			imageRef,
		),
	)
	Expect(err).ToNot(HaveOccurred())

	res := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "image",
			Version: "0.1.0",
			Type:    cdv2.OCIImageType,
		},
		Relation: cdv2.LocalRelation,
		Access:   &acc,
	}

	return res
}

func createImageIndexRes(ctx context.Context) cdv2.Resource {
	imageIndexRef = testenv.Addr + "/test/downloaders/image-index:0.1.0"

	configData := []byte("config-data")
	layersData := [][]byte{
		[]byte("layer-1-data"),
		[]byte("layer-2-data"),
	}

	manifest1Desc, _ := testutils.UploadTestImage(ctx, ociClient, imageIndexRef, ocispecv1.MediaTypeImageManifest, configData, layersData)
	manifest1Desc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	manifest2Desc, _ := testutils.UploadTestImage(ctx, ociClient, imageIndexRef, ocispecv1.MediaTypeImageManifest, configData, layersData)
	manifest2Desc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "windows",
	}

	index := ocispecv1.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: []ocispecv1.Descriptor{
			manifest1Desc,
			manifest2Desc,
		},
		Annotations: map[string]string{
			"test": "test",
		},
	}

	testutils.UploadTestIndex(ctx, ociClient, imageIndexRef, ocispecv1.MediaTypeImageIndex, index)

	var err error
	ociArtifact, err := ociClient.GetOCIArtifact(ctx, imageIndexRef)
	Expect(err).ToNot(HaveOccurred())
	expectedImageIndex = *ociArtifact.GetIndex()

	acc, err := cdv2.NewUnstructured(
		cdv2.NewOCIRegistryAccess(
			imageIndexRef,
		),
	)
	Expect(err).ToNot(HaveOccurred())

	res := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "image-index",
			Version: "0.1.0",
			Type:    cdv2.OCIImageType,
		},
		Relation: cdv2.LocalRelation,
		Access:   &acc,
	}

	return res
}
