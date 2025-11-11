// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/credentials"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/testutils"
)

func RunPushAndPullImageTest(ref, manifestMediaType string) {
	ctx := context.Background()
	defer ctx.Done()

	configData := []byte("config-data")
	layersData := [][]byte{
		[]byte("layer-1-data"),
		[]byte("layer-2-data"),
	}

	manifestDesc, manifestBytes := testutils.UploadTestImage(ctx, client, ref, manifestMediaType, configData, layersData)

	testutils.CompareRemoteManifest(ctx, client, ref, manifestDesc, manifestBytes, configData, layersData)
}

func RunPushAndPullImageIndexTest(untaggedRepo, indexMediaType string) {
	ctx := context.Background()
	defer ctx.Done()

	configData1 := []byte("config-data")
	layersData1 := [][]byte{
		[]byte("layer-1-data"),
		[]byte("layer-2-data"),
	}
	_, manifest1Desc, blobMap := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData1, layersData1)
	manifest1Ref := fmt.Sprintf("%s@%s", untaggedRepo, manifest1Desc.Digest)
	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})
	manifest1Bytes := blobMap[manifest1Desc.Digest]
	Expect(client.PushRawManifest(ctx, manifest1Ref, manifest1Desc, manifest1Bytes, ociclient.WithStore(store))).To(Succeed())

	configData2 := []byte("config-data2")
	layersData2 := [][]byte{
		[]byte("layer-1-data2"),
		[]byte("layer-2-data2"),
	}
	_, manifest2Desc, blobMap := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData2, layersData2)
	manifest2Ref := fmt.Sprintf("%s@%s", untaggedRepo, manifest2Desc.Digest)
	store = ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})
	manifest2Bytes := blobMap[manifest2Desc.Digest]
	Expect(client.PushRawManifest(ctx, manifest2Ref, manifest2Desc, manifest2Bytes, ociclient.WithStore(store))).To(Succeed())

	manifest1IndexDesc := manifest1Desc
	manifest1IndexDesc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	manifest2IndexDesc := manifest2Desc
	manifest2IndexDesc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "windows",
	}

	index := ocispecv1.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: []ocispecv1.Descriptor{
			manifest1IndexDesc,
			manifest2IndexDesc,
		},
		Annotations: map[string]string{
			"test": "test",
		},
	}

	multiArchRef := untaggedRepo + ":v0.1.0"
	indexDesc, indexBytes := testutils.UploadTestIndex(ctx, client, multiArchRef, indexMediaType, index)

	actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, multiArchRef)
	Expect(err).ToNot(HaveOccurred())
	Expect(actualIndexDesc).To(Equal(indexDesc))
	Expect(actualIndexBytes).To(Equal(indexBytes))

	testutils.CompareRemoteManifest(ctx, client, manifest1Ref, manifest1Desc, manifest1Bytes, configData1, layersData1)
	testutils.CompareRemoteManifest(ctx, client, manifest2Ref, manifest2Desc, manifest2Bytes, configData2, layersData2)
}

var _ = Describe("client", func() {

	Context("Client", func() {

		It("should push and pull a single architecture image without modifications (oci media type)", func() {
			ref := fmt.Sprintf("%s/%s", testenv.Addr, "single-arch-tests/0/artifact:v0.0.1")
			RunPushAndPullImageTest(ref, ocispecv1.MediaTypeImageManifest)
		}, 20)

		It("should push and pull a multi architecture image without modifications (oci media type)", func() {
			untaggedRef := fmt.Sprintf("%s/%s", testenv.Addr, "multi-arch-tests/0/artifact")
			RunPushAndPullImageIndexTest(untaggedRef, ocispecv1.MediaTypeImageIndex)
		}, 20)

		// TODO: investigate why this test isn't working (could be registry not accepting docker media type)
		// It("should push and pull a single architecture image without modifications (docker media type)", func() {
		// 	RunPushAndPullTest("single-arch-tests/1/artifact:0.0.1", images.MediaTypeDockerSchema2Manifest)
		// }, 20)

		// TODO: investigate why this test isn't working (could be registry not accepting docker media type)
		// It("should push and pull a multi architecture image without modifications (docker media type)", func() {
		// 	RunPushAndPullImageIndexTest("multi-arch-tests/1/artifact", images.MediaTypeDockerSchema2ManifestList)
		// }, 20)

		It("should push and pull an empty oci image index", func() {
			ctx := context.Background()
			defer ctx.Done()

			ref := testenv.Addr + "/multi-arch-tests/2/empty-img:v0.0.1"
			index := ocispecv1.Index{
				Versioned: specs.Versioned{
					SchemaVersion: 2,
				},
				Manifests: []ocispecv1.Descriptor{},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			indexBytes, err := json.Marshal(index)
			Expect(err).ToNot(HaveOccurred())

			indexDesc := ocispecv1.Descriptor{
				MediaType: ocispecv1.MediaTypeImageIndex,
				Digest:    digest.FromBytes(indexBytes),
				Size:      int64(len(indexBytes)),
			}

			store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(indexBytes)
				return err
			})

			Expect(client.PushRawManifest(ctx, ref, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, ref)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))
		}, 20)

		It("should push and pull an oci image index with only 1 manifest and no platform information", func() {
			ctx := context.Background()
			defer ctx.Done()

			configData := []byte("config-data")
			layersData := [][]byte{
				[]byte("layer-1-data"),
				[]byte("layer-2-data"),
			}
			untaggedRef := testenv.Addr + "/multi-arch-tests/3/img"

			_, manifest1Desc, blobMap := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData, layersData)
			manifest1Ref := fmt.Sprintf("%s@%s", untaggedRef, manifest1Desc.Digest)
			store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(blobMap[desc.Digest])
				return err
			})
			manifest1Bytes := blobMap[manifest1Desc.Digest]
			Expect(client.PushRawManifest(ctx, manifest1Ref, manifest1Desc, manifest1Bytes, ociclient.WithStore(store))).To(Succeed())

			index := ocispecv1.Index{
				Versioned: specs.Versioned{
					SchemaVersion: 2,
				},
				Manifests: []ocispecv1.Descriptor{
					manifest1Desc,
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			indexBytes, err := json.Marshal(index)
			Expect(err).ToNot(HaveOccurred())

			indexDesc := ocispecv1.Descriptor{
				MediaType: ocispecv1.MediaTypeImageIndex,
				Digest:    digest.FromBytes(indexBytes),
				Size:      int64(len(indexBytes)),
			}

			store = ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(indexBytes)
				return err
			})

			multiArchRef := untaggedRef + ":v0.1.0"
			Expect(client.PushRawManifest(ctx, multiArchRef, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, multiArchRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))

			testutils.CompareRemoteManifest(ctx, client, manifest1Ref, manifest1Desc, manifest1Bytes, configData, layersData)
		}, 20)

		It("should copy an oci artifact", func() {
			ctx := context.Background()
			defer ctx.Done()

			configData := []byte("config-data")
			layersData := [][]byte{
				[]byte("layer-1-data"),
				[]byte("layer-2-data"),
			}
			ref := testenv.Addr + "/single-arch-tests/2/src/artifact:v0.0.1"
			mdesc, mbytes := testutils.UploadTestImage(ctx, client, ref, ocispecv1.MediaTypeImageManifest, configData, layersData)
			newRef := testenv.Addr + "/single-arch-tests/2/tgt/artifact:v0.0.1"

			Expect(ociclient.Copy(ctx, client, ref, newRef)).To(Succeed())

			testutils.CompareRemoteManifest(ctx, client, newRef, mdesc, mbytes, configData, layersData)
		}, 20)

		It("should copy an oci image index", func() {
			ctx := context.Background()
			defer ctx.Done()

			untaggedSrcRef := testenv.Addr + "/multi-arch-tests/4/src/img"
			untaggedTgtRef := testenv.Addr + "/multi-arch-tests/4/tgt/img"

			configData := []byte("config-data")
			layersData := [][]byte{
				[]byte("layer-1-data"),
				[]byte("layer-2-data"),
			}
			_, manifest1Desc, blobMap := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData, layersData)
			manifest1Ref := fmt.Sprintf("%s@%s", untaggedSrcRef, manifest1Desc.Digest)
			store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(blobMap[desc.Digest])
				return err
			})
			manifest1Bytes := blobMap[manifest1Desc.Digest]
			Expect(client.PushRawManifest(ctx, manifest1Ref, manifest1Desc, manifest1Bytes, ociclient.WithStore(store))).To(Succeed())

			configData2 := []byte("config-data2")
			layersData2 := [][]byte{
				[]byte("layer-1-data2"),
				[]byte("layer-2-data2"),
			}
			_, manifest2Desc, blobMap := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData2, layersData2)
			manifest2Ref := fmt.Sprintf("%s@%s", untaggedSrcRef, manifest2Desc.Digest)
			store = ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(blobMap[desc.Digest])
				return err
			})
			manifest2Bytes := blobMap[manifest2Desc.Digest]
			Expect(client.PushRawManifest(ctx, manifest2Ref, manifest2Desc, manifest2Bytes, ociclient.WithStore(store))).To(Succeed())

			manifest1IndexDesc := manifest1Desc
			manifest1IndexDesc.Platform = &ocispecv1.Platform{
				Architecture: "amd64",
				OS:           "linux",
			}

			manifest2IndexDesc := manifest2Desc
			manifest2IndexDesc.Platform = &ocispecv1.Platform{
				Architecture: "amd64",
				OS:           "windows",
			}

			index := ocispecv1.Index{
				Versioned: specs.Versioned{SchemaVersion: 2},
				Manifests: []ocispecv1.Descriptor{
					manifest1IndexDesc,
					manifest2IndexDesc,
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			multiArchSrcRef := untaggedSrcRef + ":v0.1.0"
			indexDesc, indexBytes := testutils.UploadTestIndex(ctx, client, multiArchSrcRef, ocispecv1.MediaTypeImageIndex, index)

			multiArchTgtRef := untaggedTgtRef + ":v0.0.1"
			manifest1TgtRef := fmt.Sprintf("%s@%s", untaggedTgtRef, manifest1Desc.Digest)
			manifest2TgtRef := fmt.Sprintf("%s@%s", untaggedTgtRef, manifest2Desc.Digest)

			Expect(ociclient.Copy(ctx, client, multiArchSrcRef, multiArchTgtRef)).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, multiArchTgtRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))

			testutils.CompareRemoteManifest(ctx, client, manifest1TgtRef, manifest1Desc, manifest1Bytes, configData, layersData)
			testutils.CompareRemoteManifest(ctx, client, manifest2TgtRef, manifest2Desc, manifest2Bytes, configData2, layersData2)
		}, 20)

	})

	Context("ExtendedClient", func() {
		Context("ListTags", func() {
			var (
				server  *httptest.Server
				host    string
				handler func(http.ResponseWriter, *http.Request)
				makeRef = func(repo string) string {
					return fmt.Sprintf("%s/%s", host, repo)
				}
			)

			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					handler(writer, request)
				}))

				hostUrl, err := url.Parse(server.URL)
				Expect(err).ToNot(HaveOccurred())
				host = hostUrl.Host
			})

			AfterEach(func() {
				server.Close()
			})

			It("should return a list of tags", func() {
				var (
					ctx        = context.Background()
					repository = "myproject/repo/myimage"
				)
				defer ctx.Done()
				handler = func(w http.ResponseWriter, req *http.Request) {
					if req.URL.Path == "/v2/" {
						// first auth discovery call by the library
						w.WriteHeader(200)
						return
					}
					Expect(req.URL.String()).To(Equal("/v2/myproject/repo/myimage/tags/list?n=1000"))
					w.WriteHeader(200)
					_, _ = w.Write([]byte(`
{
  "tags": [ "0.0.1", "0.0.2" ]
}
`))
				}

				client, err := ociclient.NewClient(logr.Discard(),
					ociclient.AllowPlainHttp(true),
					ociclient.WithKeyring(credentials.New()))
				Expect(err).ToNot(HaveOccurred())
				tags, err := client.ListTags(ctx, makeRef(repository))
				Expect(err).ToNot(HaveOccurred())
				Expect(tags).To(ConsistOf("0.0.1", "0.0.2"))
			})

		})

		Context("ListRepositories", func() {
			var (
				server  *httptest.Server
				host    string
				handler func(http.ResponseWriter, *http.Request)
				makeRef = func(repo string) string {
					return fmt.Sprintf("%s/%s", host, repo)
				}
			)

			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					handler(writer, request)
				}))

				hostUrl, err := url.Parse(server.URL)
				Expect(err).ToNot(HaveOccurred())
				host = hostUrl.Host
			})

			AfterEach(func() {
				server.Close()
			})

			It("should return a list of repositories", func() {
				var (
					ctx        = context.Background()
					repository = "myproject/repo"
				)
				defer ctx.Done()
				handler = func(w http.ResponseWriter, req *http.Request) {
					if req.URL.Path == "/v2/" {
						// first auth discovery call by the library
						w.WriteHeader(200)
						return
					}
					Expect(req.URL.String()).To(Equal("/v2/_catalog?n=1000"))
					w.WriteHeader(200)
					_, _ = w.Write([]byte(`
{
  "repositories": [ "myproject/repo/image1", "myproject/repo/image2" ]
}
`))
				}

				client, err := ociclient.NewClient(logr.Discard(),
					ociclient.AllowPlainHttp(true),
					ociclient.WithKeyring(credentials.New()))
				Expect(err).ToNot(HaveOccurred())
				repos, err := client.ListRepositories(ctx, makeRef(repository))
				Expect(err).ToNot(HaveOccurred())
				Expect(repos).To(ConsistOf(makeRef("myproject/repo/image1"), makeRef("myproject/repo/image2")))
			})

		})
	})

})
