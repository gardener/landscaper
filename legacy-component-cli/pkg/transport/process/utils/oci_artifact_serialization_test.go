// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package utils_test

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/oci"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/testutils"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

var _ = ginkgo.Describe("oci artifact serialization", func() {

	ginkgo.Context("serialize and deserialize oci artifact", func() {

		ginkgo.It("should correctly serialize and deserialize image", func() {
			configData := []byte("config-data")
			layers := [][]byte{
				[]byte("layer-data"),
			}
			m, _, _ := testutils.CreateImage(ocispecv1.MediaTypeImageManifest, configData, layers)

			expectedOciArtifact, err := oci.NewManifestArtifact(
				&oci.Manifest{
					Data: m,
				},
			)
			Expect(err).ToNot(HaveOccurred())

			serializeCache := cache.NewInMemoryCache()
			Expect(serializeCache.Add(m.Config, io.NopCloser(bytes.NewReader(configData)))).To(Succeed())
			Expect(serializeCache.Add(m.Layers[0], io.NopCloser(bytes.NewReader(layers[0])))).To(Succeed())

			serializedReader, err := utils.SerializeOCIArtifact(*expectedOciArtifact, serializeCache)
			Expect(err).ToNot(HaveOccurred())

			deserializeCache := cache.NewInMemoryCache()
			actualOciArtifact, err := utils.DeserializeOCIArtifact(serializedReader, deserializeCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact.GetManifest().Data).To(Equal(expectedOciArtifact.GetManifest().Data))

			actualConfigReader, err := deserializeCache.Get(actualOciArtifact.GetManifest().Data.Config)
			Expect(err).ToNot(HaveOccurred())
			actualConfigBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualConfigBuf, actualConfigReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfigBuf.Bytes()).To(Equal(configData))

			actualLayerReader, err := deserializeCache.Get(actualOciArtifact.GetManifest().Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			actualLayerBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualLayerBuf, actualLayerReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualLayerBuf.Bytes()).To(Equal(layers[0]))
		})

		ginkgo.It("should correctly serialize and deserialize image index", func() {
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
							Data: m1,
						},
						{
							Data: m2,
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

			deserializeCache := cache.NewInMemoryCache()
			actualOciArtifact, err := utils.DeserializeOCIArtifact(serializedReader, deserializeCache)
			Expect(err).ToNot(HaveOccurred())

			// check image index and manifests
			Expect(actualOciArtifact.GetIndex().Annotations).To(Equal(expectedOciArtifact.GetIndex().Annotations))
			Expect(actualOciArtifact.GetIndex().Manifests[0].Data).To(Equal(m1))
			Expect(actualOciArtifact.GetIndex().Manifests[1].Data).To(Equal(m2))

			// check first manifest config and layer
			actualConfigReader, err := deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[0].Data.Config)
			Expect(err).ToNot(HaveOccurred())
			actualConfigBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualConfigBuf, actualConfigReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfigBuf.Bytes()).To(Equal(configData1))

			actualLayerReader, err := deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[0].Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			actualLayerBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualLayerBuf, actualLayerReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualLayerBuf.Bytes()).To(Equal(layers1[0]))

			// check second manifest config and layer
			actualConfigReader, err = deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[1].Data.Config)
			Expect(err).ToNot(HaveOccurred())
			actualConfigBuf = bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualConfigBuf, actualConfigReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfigBuf.Bytes()).To(Equal(configData2))

			actualLayerReader, err = deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[1].Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			actualLayerBuf = bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualLayerBuf, actualLayerReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualLayerBuf.Bytes()).To(Equal(layers2[0]))
		})

	})

	ginkgo.Context("serialize oci artifact", func() {

		ginkgo.It("should raise error if cache is nil", func() {
			_, err := utils.SerializeOCIArtifact(oci.Artifact{}, nil)
			Expect(err).To(MatchError("cache must not be nil"))
		})

	})

	ginkgo.Context("deserialize oci artifact", func() {

		ginkgo.It("should raise error if reader is nil", func() {
			_, err := utils.DeserializeOCIArtifact(nil, cache.NewInMemoryCache())
			Expect(err).To(MatchError("reader must not be nil"))
		})

		ginkgo.It("should raise error if cache is nil", func() {
			buf := bytes.NewBuffer([]byte{})
			_, err := utils.DeserializeOCIArtifact(buf, nil)
			Expect(err).To(MatchError("cache must not be nil"))
		})

		ginkgo.It("should raise error if tar archive contains unknown file", func() {
			fileName := "invalid-filename"
			fileContent := []byte("file-content")

			buf := bytes.NewBuffer([]byte{})
			tw := tar.NewWriter(buf)
			fileHeader := tar.Header{
				Name:    fileName,
				Size:    int64(len(fileContent)),
				Mode:    int64(os.ModePerm),
				ModTime: time.Now(),
			}

			Expect(tw.WriteHeader(&fileHeader)).To(Succeed())

			_, err := io.Copy(tw, bytes.NewReader(fileContent))
			Expect(err).ToNot(HaveOccurred())

			Expect(tw.Close()).To(Succeed())

			_, err = utils.DeserializeOCIArtifact(buf, cache.NewInMemoryCache())
			Expect(err).To(MatchError("unknown file " + fileName))
		})

	})

})
