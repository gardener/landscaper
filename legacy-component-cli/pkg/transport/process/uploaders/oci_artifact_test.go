// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package uploaders_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/oci"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/testutils"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/uploaders"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

var _ = Describe("ociArtifact", func() {

	Context("Process", func() {

		It("should upload and stream oci image", func() {
			acc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("my-registry.com/image:0.1.0"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "plain-text",
				},
			}
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/component-cli/test-component",
						Version: "0.1.0",
					},
					Resources: []cdv2.Resource{
						res,
					},
				},
			}
			res.Access = &acc
			expectedImageRef := targetCtx.BaseURL + "/image:0.1.0"
			configData := []byte("config-data")
			layers := [][]byte{
				[]byte("layer-data"),
			}
			m, mdesc, _ := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData, layers)

			expectedOciArtifact, err := oci.NewManifestArtifact(
				&oci.Manifest{
					Descriptor: mdesc,
					Data:       m,
				},
			)
			Expect(err).ToNot(HaveOccurred())

			serializeCache := cache.NewInMemoryCache()
			Expect(serializeCache.Add(m.Config, io.NopCloser(bytes.NewReader(configData)))).To(Succeed())
			Expect(serializeCache.Add(m.Layers[0], io.NopCloser(bytes.NewReader(layers[0])))).To(Succeed())

			serializedReader, err := utils.SerializeOCIArtifact(*expectedOciArtifact, serializeCache)
			Expect(err).ToNot(HaveOccurred())

			inProcessorMsg := bytes.NewBuffer([]byte{})
			Expect(utils.WriteProcessorMessage(cd, res, serializedReader, inProcessorMsg)).To(Succeed())
			Expect(err).ToNot(HaveOccurred())

			d, err := uploaders.NewOCIArtifactUploader(ociClient, serializeCache, targetCtx.BaseURL, false)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := utils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(cd))
			Expect(actualRes.Name).To(Equal(res.Name))
			Expect(actualRes.Version).To(Equal(res.Version))
			Expect(actualRes.Type).To(Equal(res.Type))

			ociAcc := cdv2.OCIRegistryAccess{}
			Expect(actualRes.Access.DecodeInto(&ociAcc)).To(Succeed())
			Expect(ociAcc.ImageReference).To(Equal(expectedImageRef))

			actualOciArtifact, err := utils.DeserializeOCIArtifact(resBlobReader, cache.NewInMemoryCache())
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact).To(Equal(expectedOciArtifact))
		})

		It("should upload and stream oci image index", func() {
			acc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("my-registry.com/image:0.1.0"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "plain-text",
				},
			}
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/component-cli/test-component",
						Version: "0.1.0",
					},
					Resources: []cdv2.Resource{
						res,
					},
				},
			}
			res.Access = &acc
			expectedImageRef := targetCtx.BaseURL + "/image:0.1.0"

			configData1 := []byte("config-data-1")
			layers1 := [][]byte{
				[]byte("layer-data-1"),
			}
			configData2 := []byte("config-data-2")
			layers2 := [][]byte{
				[]byte("layer-data-2"),
			}

			m1, m1Desc, _ := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData1, layers1)
			m1Desc.Platform = &ocispecv1.Platform{
				Architecture: "amd64",
				OS:           "linux",
			}

			m2, m2Desc, _ := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData2, layers2)
			m2Desc.Platform = &ocispecv1.Platform{
				Architecture: "amd64",
				OS:           "windows",
			}

			m1Bytes, err := json.Marshal(m1)
			Expect(err).ToNot(HaveOccurred())

			m2Bytes, err := json.Marshal(m2)
			Expect(err).ToNot(HaveOccurred())

			expectedOciArtifact, err := oci.NewIndexArtifact(
				&oci.Index{
					Manifests: []*oci.Manifest{
						{
							Descriptor: m1Desc,
							Data:       m1,
						},
						{
							Descriptor: m2Desc,
							Data:       m2,
						},
					},
					Annotations: map[string]string{
						"testkey": "testval",
					},
				},
			)
			Expect(err).ToNot(HaveOccurred())

			serializeCache := cache.NewInMemoryCache()
			Expect(serializeCache.Add(m1Desc, io.NopCloser(bytes.NewReader(m1Bytes)))).To(Succeed())
			Expect(serializeCache.Add(m1.Config, io.NopCloser(bytes.NewReader(configData1)))).To(Succeed())
			Expect(serializeCache.Add(m1.Layers[0], io.NopCloser(bytes.NewReader(layers1[0])))).To(Succeed())
			Expect(serializeCache.Add(m2Desc, io.NopCloser(bytes.NewReader(m2Bytes)))).To(Succeed())
			Expect(serializeCache.Add(m2.Config, io.NopCloser(bytes.NewReader(configData2)))).To(Succeed())
			Expect(serializeCache.Add(m2.Layers[0], io.NopCloser(bytes.NewReader(layers2[0])))).To(Succeed())

			serializedReader, err := utils.SerializeOCIArtifact(*expectedOciArtifact, serializeCache)
			Expect(err).ToNot(HaveOccurred())

			inProcessorMsg := bytes.NewBuffer([]byte{})
			Expect(utils.WriteProcessorMessage(cd, res, serializedReader, inProcessorMsg)).To(Succeed())
			Expect(err).ToNot(HaveOccurred())

			d, err := uploaders.NewOCIArtifactUploader(ociClient, serializeCache, targetCtx.BaseURL, false)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := utils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(cd))
			Expect(actualRes.Name).To(Equal(res.Name))
			Expect(actualRes.Version).To(Equal(res.Version))
			Expect(actualRes.Type).To(Equal(res.Type))

			ociAcc := cdv2.OCIRegistryAccess{}
			Expect(actualRes.Access.DecodeInto(&ociAcc)).To(Succeed())
			Expect(ociAcc.ImageReference).To(Equal(expectedImageRef))

			actualOciArtifact, err := utils.DeserializeOCIArtifact(resBlobReader, cache.NewInMemoryCache())
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact).To(Equal(expectedOciArtifact))
		})

		It("should return error for invalid access type", func() {
			acc, err := cdv2.NewUnstructured(cdv2.NewLocalOCIBlobAccess("sha256:123"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "plain-text",
				},
				Access: &acc,
			}
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/component-cli/test-component",
						Version: "0.1.0",
					},
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			u, err := uploaders.NewOCIArtifactUploader(ociClient, ociCache, targetCtx.BaseURL, false)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = utils.WriteProcessorMessage(cd, res, bytes.NewReader([]byte("Hello World")), b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = u.Process(context.TODO(), b1, b2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported access type"))
		})

	})

})
