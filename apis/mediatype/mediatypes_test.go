// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mediatype_test

import (
	"testing"

	"github.com/gardener/landscaper/apis/mediatype"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega/gstruct"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "mediatype Test Suite")
}

var _ = Describe("MediaType test suite", func() {

	DescribeTable("Stringer",
		func(raw string) {
			mt, err := mediatype.Parse(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(mt.String()).To(Equal(raw))
		},
		Entry("application/json", "application/json"),
		Entry("application/tar+gzip", "application/tar+gzip"),
		Entry("application/vnd.oci.null-sample.config.v1", "application/vnd.oci.null-sample.config.v1"),
	)

	It("Should be able to manually create a media type", func() {
		c := mediatype.GZipCompression
		mt := mediatype.MediaType{
			Type:              "application/json",
			CompressionFormat: &c,
		}
		Expect(mt.String()).To(Equal("application/json+gzip"))

		mt = mediatype.NewBuilder("application/json").Compression(mediatype.GZipCompression).Build()
		Expect(mt.String()).To(Equal("application/json+gzip"))
		Expect(mt.IsCompressed("")).To(BeTrue())
		Expect(mt.IsCompressed(mediatype.GZipCompression)).To(BeTrue())

		mt, err := mediatype.Parse(mt.String())
		Expect(err).ToNot(HaveOccurred())

		Expect(mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String()).
			To(Equal("application/vnd.gardener.landscaper.blueprint.layer.v1.tar+gzip"))
	})

	It("should parse a simple valid media type without compression", func() {
		mt, err := mediatype.Parse("application/json")
		Expect(err).ToNot(HaveOccurred())
		Expect(mt.Orig).To(Equal("application/json"))
		Expect(mt.Type).To(Equal("application/json"))
		Expect(mt.Format).To(Equal(mediatype.DefaultFormat))
		Expect(mt.CompressionFormat).To(BeNil())
	})

	It("should parse a simple valid media type with compression", func() {
		mt, err := mediatype.Parse("application/tar+gzip")
		Expect(err).ToNot(HaveOccurred())
		Expect(mt.Orig).To(Equal("application/tar+gzip"))
		Expect(mt.Type).To(Equal("application/tar"))
		Expect(mt.Format).To(Equal(mediatype.DefaultFormat))
		Expect(mt.Suffix).To(gstruct.PointTo(Equal("gzip")))
		Expect(mt.IsCompressed(mediatype.GZipCompression)).To(BeTrue())
	})

	It("should parse a simple valid media type without compression but with a suffix", func() {
		mt, err := mediatype.Parse("application/ld+json")
		Expect(err).ToNot(HaveOccurred())
		Expect(mt.Orig).To(Equal("application/ld+json"))
		Expect(mt.Type).To(Equal("application/ld"))
		Expect(mt.Format).To(Equal(mediatype.DefaultFormat))
		Expect(mt.Suffix).To(gstruct.PointTo(Equal("json")))
		Expect(mt.IsCompressed("")).To(BeFalse())
	})

	It("should parse a default valid config media type", func() {
		mt, err := mediatype.Parse("application/vnd.oci.null-sample.config.v1")
		Expect(err).ToNot(HaveOccurred())
		Expect(mt.Orig).To(Equal("application/vnd.oci.null-sample.config.v1"))
		Expect(mt.Type).To(Equal("application/vnd.oci.null-sample.config.v1"))
		Expect(mt.Format).To(Equal(mediatype.OCIConfigFormat))
		Expect(mt.Version).To(gstruct.PointTo(Equal("v1")))
		Expect(mt.FileFormat).To(BeNil())
		Expect(mt.CompressionFormat).To(BeNil())
	})

	It("should parse a custom valid config media type", func() {
		mt, err := mediatype.Parse("application/vnd.cncf.helm.chart.config.v1+json")
		Expect(err).ToNot(HaveOccurred())
		Expect(mt.Orig).To(Equal("application/vnd.cncf.helm.chart.config.v1+json"))
		Expect(mt.Type).To(Equal("application/vnd.cncf.helm.chart.config.v1"))
		Expect(mt.FileFormat).To(gstruct.PointTo(Equal("json")))
		Expect(mt.CompressionFormat).To(BeNil())
	})

	Context("Landscaper specifics", func() {
		It("should parse a legacy blueprint layer type", func() {
			mt, err := mediatype.Parse(mediatype.BlueprintArtifactsMediaTypeV0)
			layerMT, err := mediatype.Parse(mediatype.BlueprintArtifactsLayerMediaTypeV1 + "+gzip")
			Expect(err).ToNot(HaveOccurred())
			Expect(mt).To(Equal(layerMT))
		})

		It("should parse a legacy jsonschema layer type", func() {
			mt, err := mediatype.Parse(mediatype.JSONSchemaArtifactsMediaTypeV0)
			newMT, err := mediatype.Parse(mediatype.JSONSchemaArtifactsMediaTypeV1)
			Expect(err).ToNot(HaveOccurred())
			Expect(mt).To(Equal(newMT))
		})
	})
})
