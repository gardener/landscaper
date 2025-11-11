// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"
)

var _ = Describe("resolve", func() {

	Context("Resolve", func() {

		It("should fetch a component descriptor", func() {
			ctx := context.Background()
			ociClient := &testClient{
				getManifest: func(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
					return &ocispecv1.Manifest{
						Config: ocispecv1.Descriptor{
							MediaType: oci.ComponentDescriptorConfigMimeType,
							Digest:    digest.FromString("config"),
						},
						Layers: []ocispecv1.Descriptor{
							{
								MediaType: oci.ComponentDescriptorJSONMimeType,
								Digest:    digest.FromString("cd"),
							},
						},
					}, nil
				},
				fetch: func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
					switch desc.Digest.String() {
					case digest.FromString("config").String():
						config := oci.ComponentDescriptorConfig{
							ComponentDescriptorLayer: &oci.OciBlobRef{
								MediaType: oci.ComponentDescriptorConfigMimeType,
								Digest:    digest.FromString("cd").String(),
							},
						}
						return json.NewEncoder(writer).Encode(config)
					case digest.FromString("cd").String():
						data, err := codec.Encode(defaultComponentDescriptor("example.com/my-comp", "0.0.0"))
						if err != nil {
							return err
						}
						if _, err := io.Copy(writer, bytes.NewBuffer(data)); err != nil {
							return err
						}
						return nil
					default:
						return errors.New("unknown desc")
					}
				},
			}
			cd, err := oci.NewResolver(ociClient).Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), "example.com/my-comp", "0.0.0")
			Expect(err).ToNot(HaveOccurred())
			repoCtx := &cdv2.OCIRegistryRepository{}
			Expect(cd.GetEffectiveRepositoryContext().DecodeInto(repoCtx)).To(Succeed())
			Expect(repoCtx.BaseURL).To(Equal("example.com"), "the repository context should be injected")
		})

		It("should not fetch from the client of a cache is provided", func() {
			ctx := context.Background()
			ociCache := &testCache{
				get: func(ctx context.Context, repoCtx cdv2.OCIRegistryRepository, name, version string) (*cdv2.ComponentDescriptor, error) {
					return defaultComponentDescriptor("example.com/my-comp", "0.0.0"), nil
				},
				store: func(ctx context.Context, descriptor *cdv2.ComponentDescriptor) error {
					Expect(false).To(BeTrue(), "should not be called")
					return nil
				},
			}
			ociClient := &testClient{
				getManifest: func(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
					Expect(false).To(BeTrue(), "should not be called")
					return nil, nil
				},
				fetch: func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
					Expect(false).To(BeTrue(), "should not be called")
					return nil
				},
			}
			cd, err := oci.NewResolver(ociClient).WithCache(ociCache).Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), "example.com/my-comp", "0.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(cd.Name).To(Equal("example.com/my-comp"))
		})

		It("should store a component descriptor in the cache", func() {
			ctx := context.Background()
			storeCalled := false
			ociCache := &testCache{
				get: func(ctx context.Context, repoCtx cdv2.OCIRegistryRepository, name, version string) (*cdv2.ComponentDescriptor, error) {
					return nil, errors.New("not found")
				},
				store: func(ctx context.Context, descriptor *cdv2.ComponentDescriptor) error {
					storeCalled = true
					return nil
				},
			}
			ociClient := &testClient{
				getManifest: func(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
					return &ocispecv1.Manifest{
						Config: ocispecv1.Descriptor{
							MediaType: oci.ComponentDescriptorConfigMimeType,
							Digest:    digest.FromString("config"),
						},
						Layers: []ocispecv1.Descriptor{
							{
								MediaType: oci.ComponentDescriptorJSONMimeType,
								Digest:    digest.FromString("cd"),
							},
						},
					}, nil
				},
				fetch: func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
					switch desc.Digest.String() {
					case digest.FromString("config").String():
						config := oci.ComponentDescriptorConfig{
							ComponentDescriptorLayer: &oci.OciBlobRef{
								MediaType: oci.ComponentDescriptorConfigMimeType,
								Digest:    digest.FromString("cd").String(),
							},
						}
						return json.NewEncoder(writer).Encode(config)
					case digest.FromString("cd").String():
						data, err := codec.Encode(defaultComponentDescriptor("example.com/my-comp", "0.0.0"))
						if err != nil {
							return err
						}
						if _, err := io.Copy(writer, bytes.NewBuffer(data)); err != nil {
							return err
						}
						return nil
					default:
						return errors.New("unknown desc")
					}
				},
			}
			cd, err := oci.NewResolver(ociClient).WithCache(ociCache).Resolve(ctx, cdv2.NewOCIRegistryRepository("example.com", ""), "example.com/my-comp", "0.0.0")
			Expect(err).ToNot(HaveOccurred())

			repoCtx := &cdv2.OCIRegistryRepository{}
			Expect(cd.GetEffectiveRepositoryContext().DecodeInto(repoCtx)).To(Succeed())
			Expect(repoCtx.BaseURL).To(Equal("example.com"), "the repository context should be injected")
			Expect(storeCalled).To(BeTrue(), "the cache store function should be called")
		})

	})

})

func defaultComponentDescriptor(name, version string) *cdv2.ComponentDescriptor {
	cd := &cdv2.ComponentDescriptor{}
	cd.Name = name
	cd.Version = version
	cd.Provider = "internal"
	_ = cdv2.DefaultComponent(cd)
	return cd
}
