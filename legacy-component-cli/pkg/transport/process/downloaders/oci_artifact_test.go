// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders_test

import (
	"bytes"
	"context"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/downloaders"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

var _ = ginkgo.Describe("ociArtifact", func() {

	ginkgo.Context("Process", func() {

		ginkgo.It("should download and stream oci image", func() {
			ociImageRes := testComponent.Resources[imageResIndex]

			inProcessorMsg := bytes.NewBuffer([]byte{})
			err := utils.WriteProcessorMessage(testComponent, ociImageRes, nil, inProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			d, err := downloaders.NewOCIArtifactDownloader(ociClient, ociCache)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := utils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(testComponent))
			Expect(actualRes).To(Equal(ociImageRes))

			actualOciArtifact, err := utils.DeserializeOCIArtifact(resBlobReader, ociCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(*actualOciArtifact.GetManifest()).To(Equal(expectedImageManifest))
		})

		ginkgo.It("should download and stream oci image index", func() {
			ociImageIndexRes := testComponent.Resources[imageIndexResIndex]

			inProcessorMsg := bytes.NewBuffer([]byte{})
			err := utils.WriteProcessorMessage(testComponent, ociImageIndexRes, nil, inProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			d, err := downloaders.NewOCIArtifactDownloader(ociClient, ociCache)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := utils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(testComponent))
			Expect(actualRes).To(Equal(ociImageIndexRes))

			actualOciArtifact, err := utils.DeserializeOCIArtifact(resBlobReader, ociCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact.GetIndex()).To(Equal(&expectedImageIndex))
		})

		ginkgo.It("should return error if called with resource of invalid type", func() {
			localOciBlobRes := testComponent.Resources[localOciBlobResIndex]

			d, err := downloaders.NewOCIArtifactDownloader(ociClient, ociCache)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = utils.WriteProcessorMessage(testComponent, localOciBlobRes, nil, b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), b1, b2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported access type"))
		})

	})

})
