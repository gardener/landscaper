// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/cdutils"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
)

var _ = Describe("resource utils", func() {

	Context("#GetImageReferenceFromList", func() {
		It("should return the image from a component descriptor list", func() {
			data, err := ioutil.ReadFile("../../../../language-independent/test-resources/component_descriptor_v2.yaml")
			Expect(err).ToNot(HaveOccurred())
			cd := cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, &cd)).To(Succeed())

			imageAccess, err := cdutils.GetImageReferenceFromList(
				&cdv2.ComponentDescriptorList{Components: []cdv2.ComponentDescriptor{cd}},
				"github.com/gardener/gardener", "apiserver")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageAccess).To(Equal("eu.gcr.io/gardener-project/gardener/apiserver:v1.7.4"))
		})

		It("should return an error if no component matches the given name", func() {
			data, err := ioutil.ReadFile("../../../../language-independent/test-resources/component_descriptor_v2.yaml")
			Expect(err).ToNot(HaveOccurred())
			cd := cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, &cd)).To(Succeed())

			_, err = cdutils.GetImageReferenceFromList(
				&cdv2.ComponentDescriptorList{Components: []cdv2.ComponentDescriptor{cd}},
				"github.com/gardener/nocomp", "apiserver")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("#GetImageReferenceByName", func() {
		It("should return the image from a component descriptor", func() {
			data, err := ioutil.ReadFile("../../../../language-independent/test-resources/component_descriptor_v2.yaml")
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			imageAccess, err := cdutils.GetImageReferenceByName(cd, "apiserver")
			Expect(err).ToNot(HaveOccurred())

			Expect(imageAccess).To(Equal("eu.gcr.io/gardener-project/gardener/apiserver:v1.7.4"))
		})

		It("should return an error if no resource matches the given name", func() {
			data, err := ioutil.ReadFile("../../../../language-independent/test-resources/component_descriptor_v2.yaml")
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			_, err = cdutils.GetImageReferenceByName(cd, "noimage")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("#ParseImageReference", func() {
		It("should return the repository and tag", func() {
			repo, tag, seperator, err := cdutils.ParseImageReference("eu.gcr.io/gardener-project/gardener/apiserver:v1.7.4")
			Expect(err).ToNot(HaveOccurred())

			Expect(repo).To(Equal("eu.gcr.io/gardener-project/gardener/apiserver"))
			Expect(tag).To(Equal("v1.7.4"))
			Expect(seperator).To(Equal(":"))
		})

		It("should return the repository and tag - image reference contains port", func() {
			repo, tag, seperator, err := cdutils.ParseImageReference("eu.gcr.io:5000/gardener-project/gardener/apiserver:v1.7.4")
			Expect(err).ToNot(HaveOccurred())

			Expect(repo).To(Equal("eu.gcr.io:5000/gardener-project/gardener/apiserver"))
			Expect(tag).To(Equal("v1.7.4"))
			Expect(seperator).To(Equal(":"))
		})

		It("should return the repository and tag - image reference contains a SHA256", func() {
			repo, sha, seperator, err := cdutils.ParseImageReference("eu.gcr.io/gardener-project/apiserver@sha256:12345")
			Expect(err).ToNot(HaveOccurred())

			Expect(repo).To(Equal("eu.gcr.io/gardener-project/apiserver"))
			Expect(sha).To(Equal("sha256:12345"))
			Expect(seperator).To(Equal("@"))
		})

		It("should return the repository and tag - image reference contains a SHA256 and port", func() {
			repo, sha, seperator, err := cdutils.ParseImageReference("eu.gcr.io:5000/gardener-project/apiserver@sha256:12345")
			Expect(err).ToNot(HaveOccurred())

			Expect(repo).To(Equal("eu.gcr.io:5000/gardener-project/apiserver"))
			Expect(sha).To(Equal("sha256:12345"))
			Expect(seperator).To(Equal("@"))
		})

		It("should return an error - the image reference is invalid", func() {
			_, _, _, err := cdutils.ParseImageReference("eu.gcr.io/gardener-project/gardenerapiserver--v1.7.4")
			Expect(err).To(HaveOccurred())
		})
	})
})
