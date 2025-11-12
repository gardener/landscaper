// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
)

// UploadTestImage uploads an oci image manifest to a registry
func UploadTestImage(ctx context.Context, client ociclient.Client, ref, manifestMediaType string, configData []byte, layersData [][]byte) (ocispecv1.Descriptor, []byte) {
	_, desc, blobMap := CreateImage(manifestMediaType, configData, layersData)

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})

	manifestBytes := blobMap[desc.Digest]
	Expect(client.PushRawManifest(ctx, ref, desc, manifestBytes, ociclient.WithStore(store))).To(Succeed())

	return desc, manifestBytes
}

// UploadTestIndex uploads an oci image index to a registry
func UploadTestIndex(ctx context.Context, client ociclient.Client, ref, indexMediaType string, index ocispecv1.Index) (ocispecv1.Descriptor, []byte) {
	indexBytes, err := json.Marshal(index)
	Expect(err).ToNot(HaveOccurred())

	indexDesc := ocispecv1.Descriptor{
		MediaType: indexMediaType,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(indexBytes)
		return err
	})

	Expect(client.PushRawManifest(ctx, ref, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

	return indexDesc, indexBytes
}

// CreateImage creates an oci image manifest.
func CreateImage(manifestMediaType string, configData []byte, layersData [][]byte) (*ocispecv1.Manifest, ocispecv1.Descriptor, map[digest.Digest][]byte) {
	blobMap := map[digest.Digest][]byte{}

	configDesc := ocispecv1.Descriptor{
		MediaType: "text/plain",
		Digest:    digest.FromBytes(configData),
		Size:      int64(len(configData)),
	}
	blobMap[configDesc.Digest] = configData

	layerDescs := []ocispecv1.Descriptor{}
	for _, layerData := range layersData {
		layerDesc := ocispecv1.Descriptor{
			MediaType: "text/plain",
			Digest:    digest.FromBytes(layerData),
			Size:      int64(len(layerData)),
		}
		blobMap[layerDesc.Digest] = layerData
		layerDescs = append(layerDescs, layerDesc)
	}

	manifest := ocispecv1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: configDesc,
		Layers: layerDescs,
	}

	manifestBytes, err := json.Marshal(manifest)
	Expect(err).ToNot(HaveOccurred())

	manifestDesc := ocispecv1.Descriptor{
		MediaType: manifestMediaType,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}
	blobMap[manifestDesc.Digest] = manifestBytes

	return &manifest, manifestDesc, blobMap
}

func CompareRemoteManifest(ctx context.Context, client ociclient.Client, ref string, expectedManifestDesc ocispecv1.Descriptor, expectedManifestBytes []byte, expectedCfgBytes []byte, expectedLayers [][]byte) {
	actualManifestDesc, actualManifestBytes, err := client.GetRawManifest(ctx, ref)
	Expect(err).ToNot(HaveOccurred())
	Expect(actualManifestDesc).To(Equal(expectedManifestDesc))
	Expect(actualManifestBytes).To(Equal(expectedManifestBytes))

	actualManifest := ocispecv1.Manifest{}
	Expect(json.Unmarshal(actualManifestBytes, &actualManifest)).To(Succeed())

	actualConfigBuf := bytes.NewBuffer([]byte{})
	Expect(client.Fetch(ctx, ref, actualManifest.Config, actualConfigBuf)).To(Succeed())
	Expect(actualConfigBuf.Bytes()).To(Equal(expectedCfgBytes))

	for i, layerDesc := range actualManifest.Layers {
		actualLayerBuf := bytes.NewBuffer([]byte{})
		Expect(client.Fetch(ctx, ref, layerDesc, actualLayerBuf)).To(Succeed())
		Expect(actualLayerBuf.Bytes()).To(Equal(expectedLayers[i]))
	}
}
