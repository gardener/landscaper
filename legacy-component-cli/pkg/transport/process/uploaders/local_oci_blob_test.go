// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package uploaders_test

import (
	"bytes"
	"context"
	"io"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/uploaders"
	processutils "github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

var _ = ginkgo.Describe("localOciBlob", func() {

	ginkgo.Context("Process", func() {

		ginkgo.It("should upload and stream resource", func() {
			resBytes := []byte("Hello World")
			expectedDigest := digest.FromBytes(resBytes)
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

			inProcessorMsg := bytes.NewBuffer([]byte{})
			Expect(processutils.WriteProcessorMessage(cd, res, bytes.NewReader(resBytes), inProcessorMsg)).To(Succeed())

			u, err := uploaders.NewLocalOCIBlobUploader(ociClient, *targetCtx)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = u.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := processutils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(cd))
			Expect(actualRes.Name).To(Equal(res.Name))
			Expect(actualRes.Version).To(Equal(res.Version))
			Expect(actualRes.Type).To(Equal(res.Type))

			acc := cdv2.LocalOCIBlobAccess{}
			Expect(actualRes.Access.DecodeInto(&acc)).To(Succeed())
			Expect(acc.Digest).To(Equal(string(expectedDigest)))

			resBlob := bytes.NewBuffer([]byte{})
			_, err = io.Copy(resBlob, resBlobReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(resBlob.Bytes()).To(Equal(resBytes))

			desc := ocispecv1.Descriptor{
				Digest: expectedDigest,
				Size:   int64(len(resBytes)),
			}
			buf := bytes.NewBuffer([]byte{})
			Expect(ociClient.Fetch(context.TODO(), utils.CalculateBlobUploadRef(*targetCtx, cd.Name, cd.Version), desc, buf)).To(Succeed())
			Expect(buf.Bytes()).To(Equal(resBytes))
		})

		ginkgo.It("should return error if resource blob is nil", func() {
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
			cd := cdv2.ComponentDescriptor{}

			u, err := uploaders.NewLocalOCIBlobUploader(ociClient, *targetCtx)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = processutils.WriteProcessorMessage(cd, res, nil, b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = u.Process(context.TODO(), b1, b2)
			Expect(err).To(MatchError("resource blob must not be nil"))
		})

	})

})
