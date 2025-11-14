// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/go-logr/logr"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

var _ = ginkgo.Describe("Manifest v1 Conversion", func() {

	ginkgo.Context("Create Config", func() {

		ginkgo.It("should create oci config", func() {
			v1layer0Digest := digest.FromBytes([]byte("v1_layer_0"))
			v1layer1Digest := digest.FromBytes([]byte("v1_layer_1"))

			// layers in v1 and v2 are reversed
			diffIDs := []digest.Digest{
				v1layer1Digest,
				v1layer0Digest,
			}

			fslayers := []ociclient.FSLayer{
				{
					BlobSum: v1layer0Digest,
				},
				{
					BlobSum: v1layer1Digest,
				},
			}

			v1History := []ociclient.History{
				{
					V1Compatibility: `
{
	"id": "v1_layer_0",
	"container_config": {
		"Env": [
			"MY_ENV=test-val"
		],
		"Cmd": [
			"echo",
			"v1_layer_0"
		],
		"Entrypoint": [
			"/my-entrypoint"
		]
	},
	"architecture": "amd64",
	"os": "linux"
}
`,
				},
				{
					V1Compatibility: `
{
	"id": "v1_layer_1",
	"container_config": {
		"Cmd": [
			"echo",
			"v1_layer_1"
		]
	}
}
`,
				},
			}

			v2History := []ocispecv1.History{}
			for i := len(v1History) - 1; i >= 0; i-- {
				v1h := v1History[i]

				marshaledV1h, err := json.Marshal(v1h)
				Expect(err).ToNot(HaveOccurred())

				v2h := ocispecv1.History{}
				Expect(json.Unmarshal(marshaledV1h, &v2h)).ToNot(HaveOccurred())

				v2History = append(v2History, v2h)
			}

			expectedCfg := ocispecv1.Image{
				Platform: ocispecv1.Platform{
					Architecture: "amd64",
					OS:           "linux",
				},
				Config:  ocispecv1.ImageConfig{},
				History: v2History,
				RootFS: ocispecv1.RootFS{
					Type:    "layers",
					DiffIDs: diffIDs,
				},
			}

			v1Manifest := ociclient.V1Manifest{
				FSLayers: fslayers,
				History:  v1History,
			}

			actualCfgDesc, actualCfgBytes, err := ociclient.CreateV2Config(&v1Manifest, diffIDs, v2History)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualCfgDesc.MediaType).To(Equal(ocispecv1.MediaTypeImageConfig))
			Expect(actualCfgDesc.Digest).To(Equal(digest.FromBytes(actualCfgBytes)))
			Expect(actualCfgDesc.Size).To(Equal(int64(len(actualCfgBytes))))

			actualCfg := ocispecv1.Image{}
			Expect(json.Unmarshal(actualCfgBytes, &actualCfg)).ToNot(HaveOccurred())
			Expect(actualCfg).To(Equal(expectedCfg))
		}, 20)

	})

	ginkgo.Context("Parse V1 Manifest", func() {

		var (
			server      *httptest.Server
			host        string
			handler     func(http.ResponseWriter, *http.Request)
			makeBlobUrl = func(repo string, blobDigest digest.Digest) string {
				return fmt.Sprintf("/v2/%s/blobs/%s", repo, blobDigest)
			}
			makeRef = func(repo string) string {
				return fmt.Sprintf("%s/%s", host, repo)
			}
		)

		ginkgo.BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				handler(writer, request)
			}))

			hostUrl, err := url.Parse(server.URL)
			Expect(err).ToNot(HaveOccurred())
			host = hostUrl.Host
		})

		ginkgo.AfterEach(func() {
			server.Close()
		})

		ginkgo.It("should parse V1 manifest", func() {
			decompressedV1Layer0 := []byte("v1_layer_0")
			decompressedV1Layer1 := []byte("v1_layer_1")
			decompressedV1Layer2 := []byte("v1_layer_2")
			decompressedV1Layer3 := []byte("")

			decompressedV1layer0Digest := digest.FromBytes(decompressedV1Layer0)
			decompressedV1layer1Digest := digest.FromBytes(decompressedV1Layer1)
			decompressedV1layer2Digest := digest.FromBytes(decompressedV1Layer2)
			decompressedV1layer3Digest := digest.FromBytes(decompressedV1Layer3)

			compressedV1Layer0, err := utils.Gzip(decompressedV1Layer0, gzip.BestCompression)
			Expect(err).ToNot(HaveOccurred())

			compressedV1Layer0Digest := digest.FromBytes(compressedV1Layer0)

			// layers in v1 and v2 are reversed
			expectedDiffIDs := []digest.Digest{
				decompressedV1layer2Digest,
				decompressedV1layer0Digest,
			}

			expectedLayers := []ocispecv1.Descriptor{
				{
					MediaType: ocispecv1.MediaTypeImageLayer,
					Size:      int64(len(decompressedV1Layer2)),
					Digest:    decompressedV1layer2Digest,
				},
				{
					MediaType: ocispecv1.MediaTypeImageLayerGzip,
					Size:      int64(len(compressedV1Layer0)),
					Digest:    compressedV1Layer0Digest,
				},
			}

			expectedHistory := []ocispecv1.History{
				{
					Created:    &time.Time{},
					CreatedBy:  "",
					EmptyLayer: true,
				},
				{
					Created:   &time.Time{},
					CreatedBy: "echo v1_layer_2",
				},
				{
					Created:    &time.Time{},
					CreatedBy:  "echo v1_layer_1",
					EmptyLayer: true,
				},
				{
					Created:   &time.Time{},
					CreatedBy: "echo v1_layer_0",
				},
			}

			repo := "my-repo"
			ref := makeRef(repo)

			client, err := ociclient.NewClient(
				logr.Discard(),
				ociclient.AllowPlainHttp(true),
				ociclient.WithCache(cache.NewInMemoryCache()),
			)
			Expect(err).ToNot(HaveOccurred())

			fslayers := []ociclient.FSLayer{
				{
					BlobSum: compressedV1Layer0Digest,
				},
				{
					BlobSum: decompressedV1layer1Digest,
				},
				{
					BlobSum: decompressedV1layer2Digest,
				},
				{
					BlobSum: decompressedV1layer3Digest,
				},
			}

			history := []ociclient.History{
				{
					V1Compatibility: `
{
	"id": "v1_layer_0",
	"container_config": {
		"Env": [
			"MY_ENV=test-val"
		],
		"Cmd": [
			"echo",
			"v1_layer_0"
		],
		"Entrypoint": [
			"/my-entrypoint"
		]
	},
	"architecture": "amd64",
	"os": "linux"
}
`,
				},
				{
					V1Compatibility: `
{
	"id": "v1_layer_1",
	"container_config": {
		"Cmd": [
			"echo",
			"v1_layer_1"
		]
	},
	"throwAway": true
}
`,
				},
				{
					V1Compatibility: `
{
	"id": "v1_layer_2",
	"container_config": {
		"Cmd": [
			"echo",
			"v1_layer_2"
		]
	}
}
`,
				},
				{
					V1Compatibility: `
{
	"id": "v1_layer_3",
	"container_config": {
		"Cmd": []
	},
	"Size": 0
}
`,
				},
			}

			v1Manifest := ociclient.V1Manifest{
				FSLayers: fslayers,
				History:  history,
			}

			handler = func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/v2/" {
					// first auth discovery call by the library
					w.WriteHeader(200)
					return
				}

				var data []byte
				switch req.URL.String() {
				case makeBlobUrl(repo, compressedV1Layer0Digest):
					data = compressedV1Layer0
				case makeBlobUrl(repo, decompressedV1layer1Digest):
					data = decompressedV1Layer1
				case makeBlobUrl(repo, decompressedV1layer2Digest):
					data = decompressedV1Layer2
				}

				w.WriteHeader(200)
				_, _ = w.Write([]byte(data))
			}

			actualLayers, actualDiffIDs, actualHistory, err := ociclient.ParseV1Manifest(
				context.TODO(),
				client,
				ref,
				&v1Manifest,
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualLayers).To(Equal(expectedLayers))
			Expect(actualDiffIDs).To(Equal(expectedDiffIDs))
			Expect(actualHistory).To(Equal(expectedHistory))
		}, 20)

	})

})
