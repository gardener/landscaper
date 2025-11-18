// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/processors"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

var _ = ginkgo.Describe("resourceLabeler", func() {

	ginkgo.Context("Process", func() {

		ginkgo.It("should correctly add labels", func() {
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}

			l1 := cdv2.Label{
				Name:  "first-label",
				Value: json.RawMessage(`"true"`),
			}
			l2 := cdv2.Label{
				Name:  "second-label",
				Value: json.RawMessage(`"true"`),
			}

			resBytes := []byte("resource-blob")

			expectedRes := res
			expectedRes.Labels = append(expectedRes.Labels, l1)
			expectedRes.Labels = append(expectedRes.Labels, l2)

			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			inBuf := bytes.NewBuffer([]byte{})
			Expect(utils.WriteProcessorMessage(cd, res, bytes.NewReader(resBytes), inBuf)).To(Succeed())

			outbuf := bytes.NewBuffer([]byte{})

			p1 := processors.NewResourceLabeler(l1, l2)
			Expect(p1.Process(context.TODO(), inBuf, outbuf)).To(Succeed())

			actualCD, actualRes, actualResBlobReader, err := utils.ReadProcessorMessage(outbuf)
			Expect(err).ToNot(HaveOccurred())

			Expect(*actualCD).To(Equal(cd))
			Expect(actualRes).To(Equal(expectedRes))

			actualResBlobBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualResBlobBuf, actualResBlobReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualResBlobBuf.Bytes()).To(Equal(resBytes))
		})

	})
})
