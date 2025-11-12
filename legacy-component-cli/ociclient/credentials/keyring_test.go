// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials_test

import (
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/credentials"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "credentials Test Suite")
}

var _ = Describe("Keyrings", func() {

	Context("#Get", func() {
		It("should parse authentication config from a dockerconfig and match the hostname", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("eu.gcr.io/my-project/myimage")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("test"))
		})

		It("should return nil if no auth match the url", func() {
			keyring, err := credentials.NewBuilder(logr.Discard()).DisableDefaultConfig().Build()
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("eu.gcr.io/my-project/myimage/")
			Expect(auth).To(BeNil())
		})

		It("should parse authentication config from a dockerconfig and match the hostname with protocol", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("docker.io")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("docker"))
		})

		It("should fallback to legacy docker domain is no secret can be found for the new one. ", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig-legacy.json"})
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("docker.io")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("legacy"))
		})

		It("should match a whole resource url", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("eu.gcr.io/my-other-config/my-test:v1.2.3")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("test"))
		})

		It("should match the hostname with a prefix", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("eu.gcr.io/my-proj/my-test:v1.2.3")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("myproj"))
		})

		It("should parse authentication config from a dockerconfig and match the reference from dockerhub", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("ubuntu:18.4")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("docker"))
		})

		It("should skip emtpy credentials if multiple are defined", func() {
			keyring, err := credentials.NewBuilder(logr.Discard()).
				FromConfigFiles("./testdata/dockerconfig-empty.json").
				FromConfigFiles("./testdata/dockerconfig.json").
				Build()
			Expect(err).ToNot(HaveOccurred())

			auth := keyring.Get("eu.gcr.io/my-project/myimage")
			Expect(auth).ToNot(BeNil())
			Expect(auth.GetUsername()).To(Equal("test"))
		})
	})

	Context("#GetCredentials", func() {
		It("should parse authentication config from a dockerconfig and match the hostname", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
			Expect(err).ToNot(HaveOccurred())

			username, _, err := keyring.GetCredentials("eu.gcr.io")
			Expect(err).ToNot(HaveOccurred())
			Expect(username).To(Equal("test"))
		})

		It("should fallback to legacy docker domain is no secret can be found for the new one. ", func() {
			keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig-legacy.json"})
			Expect(err).ToNot(HaveOccurred())

			username, _, err := keyring.GetCredentials("docker.io")
			Expect(err).ToNot(HaveOccurred())
			Expect(username).To(Equal("legacy"))
		})
	})

})
