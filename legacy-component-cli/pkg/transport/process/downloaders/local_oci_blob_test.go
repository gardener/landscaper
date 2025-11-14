// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders_test

import (
	"bytes"
	"context"
	"io"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/downloaders"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

var _ = ginkgo.Describe("localOciBlob", func() {

	ginkgo.Context("Process", func() {

		ginkgo.It("should download and stream resource", func() {
			localOciBlobRes := testComponent.Resources[localOciBlobResIndex]

			inProcessorMsg := bytes.NewBuffer([]byte{})
			err := utils.WriteProcessorMessage(testComponent, localOciBlobRes, nil, inProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			d, err := downloaders.NewLocalOCIBlobDownloader(ociClient)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := utils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(testComponent))
			Expect(actualRes).To(Equal(localOciBlobRes))

			resBlob := bytes.NewBuffer([]byte{})
			_, err = io.Copy(resBlob, resBlobReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(resBlob.Bytes()).To(Equal(localOciBlobData))
		})

		ginkgo.It("should return error if called with resource of invalid access type", func() {
			ociArtifactRes := testComponent.Resources[imageResIndex]

			d, err := downloaders.NewLocalOCIBlobDownloader(ociClient)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = utils.WriteProcessorMessage(testComponent, ociArtifactRes, nil, b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), b1, b2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported access type"))
		})

	})

})
