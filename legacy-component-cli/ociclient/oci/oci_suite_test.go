// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci_test

import (
	"testing"

	. "github.com/onsi/ginkgo/extensions/table"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/oci"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "oci Test Suite")
}

var _ = ginkgo.Describe("ref", func() {

	DescribeTable("parse oci references",
		func(ref, host, repository, tag, digest string) {
			parsed, err := oci.ParseRef(ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed.Host).To(Equal(host))
			Expect(parsed.Repository).To(Equal(repository))
			if len(tag) == 0 {
				Expect(parsed.Tag).To(BeNil())
			} else {
				Expect(parsed.Tag).To(PointTo(Equal(tag)))
			}
			if len(digest) == 0 {
				Expect(parsed.Digest).To(BeNil())
			} else {
				Expect(parsed.Digest.String()).To(Equal(digest))
			}
		},
		Entry("default tagged image", "example.com/test:0.0.1", "example.com", "test", "0.0.1", ""),
		Entry("default image with digest", "example.com/test@sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182", "example.com", "test", "", "sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182"),
		Entry("docker image", "test:0.0.1", "index.docker.io", "library/test", "0.0.1", ""),
		Entry("without version", "example.com/test", "example.com", "test", "latest", ""),
		Entry("docker image without version", "test", "index.docker.io", "library/test", "latest", ""),
		Entry("with protocol", "https://example.com/test:0.0.1", "example.com", "test", "0.0.1", ""),
	)

})
