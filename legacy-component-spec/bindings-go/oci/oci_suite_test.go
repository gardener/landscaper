// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci_test

import (
	"context"
	"io"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "oci Test Suite")
}

var _ = ginkgo.Describe("helper", func() {

	ginkgo.Context("OCIRef", func() {

		ginkgo.It("should correctly parse a repository url without a protocol and a component", func() {
			repoCtx := cdv2.OCIRegistryRepository{BaseURL: "example.com"}
			ref, err := oci.OCIRef(repoCtx, "somecomp", "v0.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal("example.com/component-descriptors/somecomp:v0.0.0"))
		})

		ginkgo.It("should correctly parse a repository url with a protocol and a component", func() {
			repoCtx := cdv2.OCIRegistryRepository{BaseURL: "http://example.com"}
			ref, err := oci.OCIRef(repoCtx, "somecomp", "v0.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal("example.com/component-descriptors/somecomp:v0.0.0"))
		})

		ginkgo.It("should correctly parse a repository url without a protocol and a port and a component", func() {
			repoCtx := cdv2.OCIRegistryRepository{BaseURL: "example.com:443"}
			ref, err := oci.OCIRef(repoCtx, "somecomp", "v0.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal("example.com:443/component-descriptors/somecomp:v0.0.0"))
		})

		ginkgo.It("should correctly parse a repository url with a protocol and a port and a component", func() {
			repoCtx := cdv2.OCIRegistryRepository{BaseURL: "http://example.com:443"}
			ref, err := oci.OCIRef(repoCtx, "somecomp", "v0.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal("example.com:443/component-descriptors/somecomp:v0.0.0"))
		})

		ginkgo.It("should correctly parse a repository url with a sha256-digest name mapping", func() {
			repoCtx := cdv2.OCIRegistryRepository{
				BaseURL:              "example.com:443",
				ComponentNameMapping: cdv2.OCIRegistryDigestMapping,
			}
			ref, err := oci.OCIRef(repoCtx, "somecomp", "v0.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(ref).To(Equal("example.com:443/e9332cb39f32d0cea12252c5512f17e54fd850cf71faa5f5f0ef09056af01add:v0.0.0"))
		})

	})

})

// testClient describes a test oci client.
type testClient struct {
	getManifest func(ctx context.Context, ref string) (*ocispecv1.Manifest, error)
	fetch       func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error
}

var _ oci.Client = &testClient{}

func (t testClient) GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
	return t.getManifest(ctx, ref)
}

func (t testClient) Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
	return t.fetch(ctx, ref, desc, writer)
}

// testCache describes a test resolve cache.
type testCache struct {
	get   func(ctx context.Context, repoCtx cdv2.OCIRegistryRepository, name, version string) (*cdv2.ComponentDescriptor, error)
	store func(ctx context.Context, descriptor *cdv2.ComponentDescriptor) error
}

var _ oci.Cache = &testCache{}

func (t testCache) Get(ctx context.Context, repoCtx cdv2.OCIRegistryRepository, name, version string) (*cdv2.ComponentDescriptor, error) {
	return t.get(ctx, repoCtx, name, version)
}

func (t testCache) Store(ctx context.Context, descriptor *cdv2.ComponentDescriptor) error {
	return t.store(ctx, descriptor)
}
